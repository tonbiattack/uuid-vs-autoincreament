package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"uuid-vs-autoincreament/internal/bench"
)

func main() {
	cfg := bench.DefaultConfig()
	bench.RegisterFlags(flag.CommandLine, &cfg)
	flag.Parse()

	if err := bench.ValidateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mysqlDB, err := sql.Open("mysql", bench.MySQLDSN(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer mysqlDB.Close()

	pgDB, err := sql.Open("pgx", bench.PGDSN(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pgDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	if err := mysqlDB.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "mysql ping failed:", err)
		os.Exit(1)
	}
	if err := pgDB.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "postgres ping failed:", err)
		os.Exit(1)
	}

	results, err := bench.RunAll(ctx, mysqlDB, pgDB, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchmark failed:", err)
		os.Exit(1)
	}
	fmt.Println(bench.FormatResults(results))
}
