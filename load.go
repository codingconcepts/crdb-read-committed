package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/codingconcepts/crdb-read-committed/pkg/database"
	"github.com/codingconcepts/ring"
	"github.com/codingconcepts/semaphore"
	"github.com/codingconcepts/throttle"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	url            string
	seedRows       int
	writePercent   int
	duration       time.Duration
	qps            int
	concurrency    int
	readCommitted  bool
	readCount      uint64
	writeCount     uint64
	readLatencies  = ring.New[time.Duration](5000)
	writeLatencies = ring.New[time.Duration](5000)

	productIDs []string
)

func main() {
	runCmd := cobra.Command{
		Use:   "run",
		Short: "Start a load test",
		Run:   handleRun,
	}
	runCmd.PersistentFlags().IntVar(&qps, "qps", 100, "number of queries to run per second")
	runCmd.PersistentFlags().IntVar(&concurrency, "concurrency", 8, "number of workers to run concurrently")
	runCmd.PersistentFlags().DurationVar(&duration, "duration", time.Minute*10, "duration of test")
	runCmd.PersistentFlags().IntVar(&writePercent, "write-percent", 10, "number of writes as a percentage of total statements")
	runCmd.PersistentFlags().BoolVar(&readCommitted, "read-committed", false, "run statements with READ COMMITTED isolation")

	initCmd := cobra.Command{
		Use:   "init",
		Short: "Initialize the database ready for a load test",
		Run:   handleInit,
	}
	initCmd.PersistentFlags().IntVar(&seedRows, "seed-rows", 1000, "number of rows to seed the database with before the load test")

	rootCmd := cobra.Command{}
	rootCmd.PersistentFlags().StringVar(&url, "url", "", "database connection string")
	rootCmd.AddCommand(&runCmd, &initCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error running root command: %v", err)
	}
}

func handleInit(cmd *cobra.Command, args []string) {
	if url == "" {
		log.Fatalf("missing --url argument")
	}

	db := database.MustConnect(url, concurrency)

	if err := database.Create(db); err != nil {
		log.Fatalf("error creating database: %v", err)
	}

	if err := database.Seed(db, seedRows); err != nil {
		log.Fatalf("error seeding database: %v", err)
	}
}

func handleRun(cmd *cobra.Command, args []string) {
	if url == "" {
		log.Fatalf("missing --url argument")
	}

	db := database.MustConnect(url, concurrency)

	var err error
	if productIDs, err = database.FetchIDs(db); err != nil {
		log.Fatalf("error fetching ids ahead of test: %v", err)
	}

	writeRate := (float64(writePercent) / 100) * float64(qps)
	readRate := float64(qps) - writeRate

	fmt.Printf(
		"Sample size:     %d products\n"+
			"Isolation level: %s\n"+
			"Reads/s:         %.0f\n"+
			"Writes/s:        %.0f\n"+
			"Workers:         %d\n",
		len(productIDs),
		lo.Ternary(readCommitted, "READ COMMITTED", "SERIALIZABLE"),
		readRate,
		writeRate,
		concurrency,
	)

	kill := make(chan struct{})
	go printLoop(kill)
	go read(db, int(readRate), kill)
	go write(db, int(writeRate), kill)

	<-time.After(duration)
	kill <- struct{}{}
}

func write(db *pgxpool.Pool, rate int, kill <-chan struct{}) {
	const stmtSelect = `SELECT price FROM product WHERE id = $1`
	const stmtUpdate = `UPDATE product SET price = $1 WHERE id = $2`

	t := throttle.New(int64(rate), time.Second)
	s := semaphore.New(concurrency)

	// Allow the reader to be cancelled when the test is finished.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-kill
		cancel()
	}()

	t.DoFor(ctx, duration, func() error {
		id := productIDs[rand.Intn(len(productIDs))]

		s.Run(func() {
			start := time.Now()
			opts := txOptions()
			err := crdbpgx.ExecuteTx(ctx, db, opts, func(tx pgx.Tx) error {
				row := tx.QueryRow(ctx, stmtSelect, id)

				var price float64
				if err := row.Scan(&price); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}

				if _, err := tx.Exec(ctx, stmtUpdate, price+1, id); err != nil {
					return fmt.Errorf("updating row: %w", err)
				}

				return nil
			})

			if err != nil {
				log.Printf("[write] error: %v", err)
			}
			writeLatencies.Add(time.Since(start))
			atomic.AddUint64(&writeCount, 1)
		})

		return nil
	})
}

func read(db *pgxpool.Pool, rate int, kill <-chan struct{}) {
	const stmt = `SELECT name, price
								FROM product
								WHERE id = $1`

	t := throttle.New(int64(rate), time.Second)
	s := semaphore.New(concurrency)

	// Allow the reader to be cancelled when the test is finished.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-kill
		cancel()
	}()

	t.DoFor(ctx, duration, func() error {
		id := productIDs[rand.Intn(len(productIDs))]

		s.Run(func() {
			start := time.Now()
			opts := txOptions()
			err := crdbpgx.ExecuteTx(ctx, db, opts, func(tx pgx.Tx) error {
				row := tx.QueryRow(ctx, stmt, id)

				var name string
				var price float64

				return row.Scan(&name, &price)
			})

			if err != nil {
				log.Printf("[read] error: %v", err)
			}
			readLatencies.Add(time.Since(start))
			atomic.AddUint64(&readCount, 1)
		})

		return nil
	})
}

func printLoop(kill <-chan struct{}) {
	end := time.Now().Add(duration)
	ticker := time.NewTicker(time.Second).C

	for {
		select {
		case <-ticker:
			// Prevent race condition whereby screen is cleared before it can be interrogated
			// because kill signal arrives late and the ticker runs one too many times.
			if time.Now().After(end) {
				return
			}

			fmt.Println("\033[H\033[2J")

			fmt.Printf("read count:        %d\n", atomic.LoadUint64(&readCount))
			reads := readLatencies.Slice()
			if len(reads) > 0 {
				fmt.Printf("avg read latency:  %dms\n", lo.Sum(reads).Milliseconds()/int64(len(reads)))
			}

			fmt.Println()

			fmt.Printf("write count:       %d\n", atomic.LoadUint64(&writeCount))
			writes := writeLatencies.Slice()
			if len(writes) > 0 {
				fmt.Printf("avg write latency: %dms\n", lo.Sum(writes).Milliseconds()/int64(len(writes)))
			}

			fmt.Println()
			fmt.Printf("time left: %s", time.Until(end).Truncate(time.Second))
		case <-kill:
			return
		}
	}
}

func txOptions() pgx.TxOptions {
	if readCommitted {
		return pgx.TxOptions{
			IsoLevel: lo.Ternary(readCommitted, pgx.ReadCommitted, pgx.Serializable),
		}
	}

	return pgx.TxOptions{}
}
