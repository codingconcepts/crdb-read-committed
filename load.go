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
	readCommitted  bool
	readCount      uint64
	writeCount     uint64
	readLatencies  = ring.New[time.Duration](1000)
	writeLatencies = ring.New[time.Duration](1000)

	productIDs []string
)

func main() {
	runCmd := cobra.Command{
		Use:   "run",
		Short: "Start a load test",
		Run:   handleRun,
	}
	runCmd.PersistentFlags().IntVar(&qps, "qps", 100, "number of queries to run per second")
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

	db := database.MustConnect(url)

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

	db := database.MustConnect(url)

	var err error
	if productIDs, err = database.FetchIDs(db); err != nil {
		log.Fatalf("error fetching ids ahead of test: %v", err)
	}

	fmt.Printf(
		"load testing will cover %d products with %s isolation",
		len(productIDs),
		lo.Ternary(readCommitted, "READ COMMITTED", "SERIALIZABLE"),
	)

	kill := make(chan struct{})
	go printLoop(kill)
	go read(db, kill)
	go write(db, kill)

	<-time.After(duration)
	kill <- struct{}{}
}

func write(db *pgxpool.Pool, kill <-chan struct{}) {
	const stmt = `UPDATE product SET name = $1, price = $2 WHERE id = $3`

	t := throttle.New(int64(qps), time.Second)
	s := semaphore.New(20)

	// Allow the reader to be cancelled when the test is finished.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-kill
		cancel()
	}()

	t.DoFor(ctx, duration, func() error {
		if rand.Intn(100) > writePercent {
			return nil
		}

		id := productIDs[rand.Intn(len(productIDs))]

		s.Run(func() {
			start := time.Now()
			opts := txOptions()
			err := crdbpgx.ExecuteTx(ctx, db, opts, func(tx pgx.Tx) error {
				name, price := database.Product()
				_, err := tx.Exec(ctx, stmt, name, price, id)

				return err
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

func read(db *pgxpool.Pool, kill <-chan struct{}) {
	const stmt = `SELECT name, price
								FROM product
								WHERE id = $1`

	t := throttle.New(int64(qps), time.Second)
	s := semaphore.New(20)

	// Allow the reader to be cancelled when the test is finished.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-kill
		cancel()
	}()

	t.DoFor(ctx, duration, func() error {
		if rand.Intn(100) < writePercent || len(productIDs) == 0 {
			return nil
		}

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
			IsoLevel: pgx.ReadCommitted,
		}
	}

	return pgx.TxOptions{}
}
