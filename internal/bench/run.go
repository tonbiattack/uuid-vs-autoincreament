package bench

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func MySQLDSN(cfg Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDB,
	)
}

func PGDSN(cfg Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.PGHost,
		cfg.PGPort,
		cfg.PGUser,
		cfg.PGPassword,
		cfg.PGDB,
	)
}

func RunAll(ctx context.Context, mysqlDB, pgDB *sql.DB, cfg Config) ([]Result, error) {
	if err := setupMySQL(ctx, mysqlDB); err != nil {
		return nil, err
	}
	if err := setupPostgres(ctx, pgDB); err != nil {
		return nil, err
	}

	results := make([]Result, 0, 5)
	r, err := benchMySQLAuto(ctx, mysqlDB, cfg.Rows, cfg.Lookups)
	if err != nil {
		return nil, err
	}
	results = append(results, r)
	r, err = benchMySQLUUIDChar(ctx, mysqlDB, cfg.Rows, cfg.Lookups)
	if err != nil {
		return nil, err
	}
	results = append(results, r)
	r, err = benchMySQLUUIDBin(ctx, mysqlDB, cfg.Rows, cfg.Lookups)
	if err != nil {
		return nil, err
	}
	results = append(results, r)
	r, err = benchPGAuto(ctx, pgDB, cfg.Rows, cfg.Lookups)
	if err != nil {
		return nil, err
	}
	results = append(results, r)
	r, err = benchPGUUID(ctx, pgDB, cfg.Rows, cfg.Lookups)
	if err != nil {
		return nil, err
	}
	results = append(results, r)
	return results, nil
}

func setupMySQL(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		"DROP TABLE IF EXISTS bench_auto",
		"DROP TABLE IF EXISTS bench_uuid_char",
		"DROP TABLE IF EXISTS bench_uuid_bin",
		`CREATE TABLE bench_auto (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			payload VARCHAR(100) NOT NULL
		) ENGINE=InnoDB`,
		`CREATE TABLE bench_uuid_char (
			id CHAR(36) NOT NULL PRIMARY KEY,
			payload VARCHAR(100) NOT NULL
		) ENGINE=InnoDB`,
		`CREATE TABLE bench_uuid_bin (
			id BINARY(16) NOT NULL PRIMARY KEY,
			payload VARCHAR(100) NOT NULL
		) ENGINE=InnoDB`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("mysql setup failed: %w", err)
		}
	}
	return nil
}

func setupPostgres(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		"DROP TABLE IF EXISTS bench_auto",
		"DROP TABLE IF EXISTS bench_uuid",
		`CREATE TABLE bench_auto (
			id BIGSERIAL PRIMARY KEY,
			payload TEXT NOT NULL
		)`,
		`CREATE TABLE bench_uuid (
			id UUID PRIMARY KEY,
			payload TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("postgres setup failed: %w", err)
		}
	}
	return nil
}

func benchMySQLAuto(ctx context.Context, db *sql.DB, rows, lookups int) (Result, error) {
	insertStmt, err := db.PrepareContext(ctx, "INSERT INTO bench_auto (payload) VALUES (?)")
	if err != nil {
		return Result{}, err
	}
	defer insertStmt.Close()

	start := time.Now()
	for i := 0; i < rows; i++ {
		if _, err := insertStmt.ExecContext(ctx, fmt.Sprintf("p-%d", i)); err != nil {
			return Result{}, err
		}
	}
	insertSec := time.Since(start).Seconds()

	ids := make([]int64, 0, rows)
	rowsRes, err := db.QueryContext(ctx, "SELECT id FROM bench_auto ORDER BY id")
	if err != nil {
		return Result{}, err
	}
	for rowsRes.Next() {
		var id int64
		if err := rowsRes.Scan(&id); err != nil {
			rowsRes.Close()
			return Result{}, err
		}
		ids = append(ids, id)
	}
	rowsRes.Close()

	sample := ids
	if len(sample) > lookups {
		sample = sample[:lookups]
	}

	selectStmt, err := db.PrepareContext(ctx, "SELECT payload FROM bench_auto WHERE id = ?")
	if err != nil {
		return Result{}, err
	}
	defer selectStmt.Close()

	start = time.Now()
	for _, id := range sample {
		var payload string
		if err := selectStmt.QueryRowContext(ctx, id).Scan(&payload); err != nil {
			return Result{}, err
		}
	}
	pointSec := time.Since(start).Seconds()

	lo, hi := int64(0), int64(0)
	if len(ids) > 0 {
		lo = ids[len(ids)/4]
		hi = ids[(len(ids)*3)/4]
	}
	start = time.Now()
	var c int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bench_auto WHERE id BETWEEN ? AND ?", lo, hi).Scan(&c); err != nil {
		return Result{}, err
	}
	rangeSec := time.Since(start).Seconds()

	return Result{
		DB:               "mysql",
		Table:            "bench_auto",
		InsertRows:       rows,
		InsertSeconds:    insertSec,
		PointLookupCount: len(sample),
		PointSeconds:     pointSec,
		RangeSeconds:     rangeSec,
	}, nil
}

func benchMySQLUUIDChar(ctx context.Context, db *sql.DB, rows, lookups int) (Result, error) {
	insertStmt, err := db.PrepareContext(ctx, "INSERT INTO bench_uuid_char (id, payload) VALUES (?, ?)")
	if err != nil {
		return Result{}, err
	}
	defer insertStmt.Close()

	ids := make([]string, rows)
	start := time.Now()
	for i := 0; i < rows; i++ {
		id := uuid.NewString()
		ids[i] = id
		if _, err := insertStmt.ExecContext(ctx, id, fmt.Sprintf("p-%d", i)); err != nil {
			return Result{}, err
		}
	}
	insertSec := time.Since(start).Seconds()

	sample := ids
	if len(sample) > lookups {
		sample = sample[:lookups]
	}
	selectStmt, err := db.PrepareContext(ctx, "SELECT payload FROM bench_uuid_char WHERE id = ?")
	if err != nil {
		return Result{}, err
	}
	defer selectStmt.Close()

	start = time.Now()
	for _, id := range sample {
		var payload string
		if err := selectStmt.QueryRowContext(ctx, id).Scan(&payload); err != nil {
			return Result{}, err
		}
	}
	pointSec := time.Since(start).Seconds()

	start = time.Now()
	rowsRes, err := db.QueryContext(ctx, "SELECT id FROM bench_uuid_char ORDER BY id LIMIT 10000")
	if err != nil {
		return Result{}, err
	}
	for rowsRes.Next() {
		var id string
		if err := rowsRes.Scan(&id); err != nil {
			rowsRes.Close()
			return Result{}, err
		}
	}
	rowsRes.Close()
	rangeSec := time.Since(start).Seconds()

	return Result{
		DB:               "mysql",
		Table:            "bench_uuid_char",
		InsertRows:       rows,
		InsertSeconds:    insertSec,
		PointLookupCount: len(sample),
		PointSeconds:     pointSec,
		RangeSeconds:     rangeSec,
	}, nil
}

func benchMySQLUUIDBin(ctx context.Context, db *sql.DB, rows, lookups int) (Result, error) {
	insertStmt, err := db.PrepareContext(ctx, "INSERT INTO bench_uuid_bin (id, payload) VALUES (?, ?)")
	if err != nil {
		return Result{}, err
	}
	defer insertStmt.Close()

	ids := make([][]byte, rows)
	start := time.Now()
	for i := 0; i < rows; i++ {
		b := UUIDToBytes(uuid.New())
		ids[i] = b
		if _, err := insertStmt.ExecContext(ctx, b, fmt.Sprintf("p-%d", i)); err != nil {
			return Result{}, err
		}
	}
	insertSec := time.Since(start).Seconds()

	sample := ids
	if len(sample) > lookups {
		sample = sample[:lookups]
	}
	selectStmt, err := db.PrepareContext(ctx, "SELECT payload FROM bench_uuid_bin WHERE id = ?")
	if err != nil {
		return Result{}, err
	}
	defer selectStmt.Close()

	start = time.Now()
	for _, id := range sample {
		var payload string
		if err := selectStmt.QueryRowContext(ctx, id).Scan(&payload); err != nil {
			return Result{}, err
		}
	}
	pointSec := time.Since(start).Seconds()

	start = time.Now()
	rowsRes, err := db.QueryContext(ctx, "SELECT id FROM bench_uuid_bin ORDER BY id LIMIT 10000")
	if err != nil {
		return Result{}, err
	}
	for rowsRes.Next() {
		var b []byte
		if err := rowsRes.Scan(&b); err != nil {
			rowsRes.Close()
			return Result{}, err
		}
	}
	rowsRes.Close()
	rangeSec := time.Since(start).Seconds()

	return Result{
		DB:               "mysql",
		Table:            "bench_uuid_bin",
		InsertRows:       rows,
		InsertSeconds:    insertSec,
		PointLookupCount: len(sample),
		PointSeconds:     pointSec,
		RangeSeconds:     rangeSec,
	}, nil
}

func benchPGAuto(ctx context.Context, db *sql.DB, rows, lookups int) (Result, error) {
	insertStmt, err := db.PrepareContext(ctx, "INSERT INTO bench_auto (payload) VALUES ($1)")
	if err != nil {
		return Result{}, err
	}
	defer insertStmt.Close()

	start := time.Now()
	for i := 0; i < rows; i++ {
		if _, err := insertStmt.ExecContext(ctx, fmt.Sprintf("p-%d", i)); err != nil {
			return Result{}, err
		}
	}
	insertSec := time.Since(start).Seconds()

	ids := make([]int64, 0, rows)
	rowsRes, err := db.QueryContext(ctx, "SELECT id FROM bench_auto ORDER BY id")
	if err != nil {
		return Result{}, err
	}
	for rowsRes.Next() {
		var id int64
		if err := rowsRes.Scan(&id); err != nil {
			rowsRes.Close()
			return Result{}, err
		}
		ids = append(ids, id)
	}
	rowsRes.Close()

	sample := ids
	if len(sample) > lookups {
		sample = sample[:lookups]
	}
	selectStmt, err := db.PrepareContext(ctx, "SELECT payload FROM bench_auto WHERE id = $1")
	if err != nil {
		return Result{}, err
	}
	defer selectStmt.Close()

	start = time.Now()
	for _, id := range sample {
		var payload string
		if err := selectStmt.QueryRowContext(ctx, id).Scan(&payload); err != nil {
			return Result{}, err
		}
	}
	pointSec := time.Since(start).Seconds()

	lo, hi := int64(0), int64(0)
	if len(ids) > 0 {
		lo = ids[len(ids)/4]
		hi = ids[(len(ids)*3)/4]
	}
	start = time.Now()
	var c int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bench_auto WHERE id BETWEEN $1 AND $2", lo, hi).Scan(&c); err != nil {
		return Result{}, err
	}
	rangeSec := time.Since(start).Seconds()

	return Result{
		DB:               "postgres",
		Table:            "bench_auto",
		InsertRows:       rows,
		InsertSeconds:    insertSec,
		PointLookupCount: len(sample),
		PointSeconds:     pointSec,
		RangeSeconds:     rangeSec,
	}, nil
}

func benchPGUUID(ctx context.Context, db *sql.DB, rows, lookups int) (Result, error) {
	insertStmt, err := db.PrepareContext(ctx, "INSERT INTO bench_uuid (id, payload) VALUES ($1, $2)")
	if err != nil {
		return Result{}, err
	}
	defer insertStmt.Close()

	ids := make([]uuid.UUID, rows)
	start := time.Now()
	for i := 0; i < rows; i++ {
		id := uuid.New()
		ids[i] = id
		if _, err := insertStmt.ExecContext(ctx, id, fmt.Sprintf("p-%d", i)); err != nil {
			return Result{}, err
		}
	}
	insertSec := time.Since(start).Seconds()

	sample := ids
	if len(sample) > lookups {
		sample = sample[:lookups]
	}
	selectStmt, err := db.PrepareContext(ctx, "SELECT payload FROM bench_uuid WHERE id = $1")
	if err != nil {
		return Result{}, err
	}
	defer selectStmt.Close()

	start = time.Now()
	for _, id := range sample {
		var payload string
		if err := selectStmt.QueryRowContext(ctx, id).Scan(&payload); err != nil {
			return Result{}, err
		}
	}
	pointSec := time.Since(start).Seconds()

	start = time.Now()
	rowsRes, err := db.QueryContext(ctx, "SELECT id FROM bench_uuid ORDER BY id LIMIT 10000")
	if err != nil {
		return Result{}, err
	}
	for rowsRes.Next() {
		var id uuid.UUID
		if err := rowsRes.Scan(&id); err != nil {
			rowsRes.Close()
			return Result{}, err
		}
	}
	rowsRes.Close()
	rangeSec := time.Since(start).Seconds()

	return Result{
		DB:               "postgres",
		Table:            "bench_uuid",
		InsertRows:       rows,
		InsertSeconds:    insertSec,
		PointLookupCount: len(sample),
		PointSeconds:     pointSec,
		RangeSeconds:     rangeSec,
	}, nil
}
