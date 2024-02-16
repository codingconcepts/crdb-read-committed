package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/codingconcepts/crdb-read-committed/pkg/database"
	"github.com/codingconcepts/semaphore"
	"github.com/codingconcepts/throttle"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
)

func main() {
	url := flag.String("url", "postgres://root@localhost:26257?sslmode=disable", "database connection string")
	qps := flag.Int64("qps", 100, "number of queries to run per second")
	concurrency := flag.Int("concurrency", 8, "number of workers to run concurrently")
	duration := flag.Duration("duration", time.Second*10, "duration of test")
	isolation := flag.String("isolation", "serializable", "isolation to use [read committed | serializable]")
	accounts := flag.Int("accounts", 100000, "number of accounts to simulate")
	selection := flag.Int("selection", 10, "number of accounts to work with")
	flag.Parse()

	r := runner{
		qps:         *qps,
		concurrency: *concurrency,
		duration:    *duration,
		isolation:   *isolation,
		accounts:    *accounts,
		selection:   *selection,
	}

	db := database.MustConnect(*url, r.concurrency)

	if err := r.deinit(db); err != nil {
		log.Fatalf("error destroying database: %v", err)
	}

	if err := r.init(db); err != nil {
		log.Fatalf("error initialising database: %v", err)
	}

	if err := r.run(db); err != nil {
		log.Fatalf("error running simulation: %v", err)
	}

	if err := r.summary(db); err != nil {
		log.Fatalf("error generating summary: %v", err)
	}
}

type runner struct {
	accounts    int
	selection   int
	duration    time.Duration
	qps         int64
	concurrency int
	isolation   string

	accountIDs   []string
	selectionIDs []string

	latencies    latencies
	requestsMade uint64
}

type latencies struct {
	mu     sync.Mutex
	values []time.Duration
}

func (l *latencies) add(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.values = append(l.values, d)
}

func (r *runner) init(db *pgxpool.Pool) error {
	if err := database.Create(db); err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	if err := database.Seed(db, r.accounts); err != nil {
		return fmt.Errorf("seeding database: %w", err)
	}

	accountIDs, err := database.FetchIDs(db, r.selection)
	if err != nil {
		return fmt.Errorf("fetching ids: %w", err)
	}

	r.accountIDs = accountIDs
	return nil
}

func (r *runner) deinit(db *pgxpool.Pool) error {
	if err := database.Drop(db); err != nil {
		log.Fatalf("error creating database: %v", err)
	}

	return nil
}

func (r *runner) run(db *pgxpool.Pool) error {
	accountIDs, err := database.FetchIDs(db, r.selection)
	if err != nil {
		log.Fatalf("error fetching ids ahead of test: %v", err)
	}
	r.selectionIDs = accountIDs

	fmt.Printf("Sample size:     %d accounts\n", len(accountIDs))
	fmt.Printf("Isolation level: %s\n", strings.ToUpper(r.isolation))
	fmt.Printf("Concurrency      %d\n", r.concurrency)

	kill := make(chan struct{})
	go r.printLoop(kill)
	go r.work(db, kill)

	<-time.After(r.duration)
	kill <- struct{}{}

	return nil
}

func (r *runner) work(db *pgxpool.Pool, kill <-chan struct{}) {
	t := throttle.New(r.qps, time.Second)
	s := semaphore.New(r.concurrency)

	// Allow the reader to be cancelled when the test is finished.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-kill
		cancel()
	}()

	t.DoFor(ctx, r.duration, func() error {
		s.Run(func() {
			ids := lo.Samples(r.selectionIDs, 2)

			start := time.Now()
			opts := txOptions(r.isolation)
			err := crdbpgx.ExecuteTx(ctx, db, opts, func(tx pgx.Tx) error {
				// Get balance from source account.
				balanceSrc, err := database.FetchBalance(ctx, tx, ids[0])
				if err != nil {
					return fmt.Errorf("fetching balance from source account: %w", err)
				}

				// Get balance from destination account.
				balanceDst, err := database.FetchBalance(ctx, tx, ids[1])
				if err != nil {
					return fmt.Errorf("fetching balance from destination account: %w", err)
				}

				// Perform transfer.
				if err = database.UpdateBalance(ctx, tx, ids[0], balanceSrc-5); err != nil {
					return fmt.Errorf("debiting source account: %w", err)
				}
				if err = database.UpdateBalance(ctx, tx, ids[1], balanceDst+5); err != nil {
					return fmt.Errorf("crediting destination account: %w", err)
				}

				return nil
			})

			if err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("error: %v", err)
			}

			r.latencies.add(time.Since(start))
			atomic.AddUint64(&r.requestsMade, 1)
		})

		return nil
	})
}

func (r *runner) summary(db *pgxpool.Pool) error {
	expectedBalanceSum := r.selection * 10000

	actualBalanceSum, err := database.FetchBalancesSum(db, r.selectionIDs)
	if err != nil {
		return fmt.Errorf("fetching balance sum: %w", err)
	}

	fmt.Println("\033[H\033[2J")
	if len(r.latencies.values) > 0 {
		fmt.Printf("avg latency:       %dms\n", lo.Sum(r.latencies.values).Milliseconds()/int64(len(r.latencies.values)))
	}

	fmt.Printf("total requests:    %d\n", r.requestsMade)
	fmt.Printf("exp total balance: %d\n", expectedBalanceSum)
	fmt.Printf("act total balance: %.f\n", actualBalanceSum)

	return nil
}

func (r *runner) printLoop(kill <-chan struct{}) {
	end := time.Now().Add(r.duration)
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
			fmt.Printf("time left: %s", time.Until(end).Truncate(time.Second))
		case <-kill:
			return
		}
	}
}

func txOptions(isolation string) pgx.TxOptions {
	switch strings.ToUpper(isolation) {
	case "READ COMMITTED":
		return pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	case "SERIALIZABLE":
		return pgx.TxOptions{IsoLevel: pgx.Serializable}
	}

	panic(fmt.Sprintf("invalid isolation level: %s", isolation))
}
