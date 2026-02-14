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
	// ベンチマークの既定設定を読み込み、CLI 引数で上書き可能にする。
	cfg := bench.DefaultConfig()
	bench.RegisterFlags(flag.CommandLine, &cfg)
	flag.Parse()

	// 実行前に最低限の入力値を検証する。
	if err := bench.ValidateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// MySQL 接続を初期化する（ドライバは blank import で登録済み）。
	mysqlDB, err := sql.Open("mysql", bench.MySQLDSN(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer mysqlDB.Close()

	// PostgreSQL 接続を初期化する（pgx stdlib ドライバを利用）。
	pgDB, err := sql.Open("pgx", bench.PGDSN(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pgDB.Close()

	// 長時間実行を想定しつつ、無限待ちを避けるため全体タイムアウトを設定する。
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// 実ベンチ前に DB 到達性を確認し、失敗時は即時終了する。
	if err := mysqlDB.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "mysql ping failed:", err)
		os.Exit(1)
	}
	if err := pgDB.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "postgres ping failed:", err)
		os.Exit(1)
	}

	// 各方式のベンチマークを順に実行し、CSV 形式で結果を出力する。
	results, err := bench.RunAll(ctx, mysqlDB, pgDB, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchmark failed:", err)
		os.Exit(1)
	}
	fmt.Println(bench.FormatResults(results))
}
