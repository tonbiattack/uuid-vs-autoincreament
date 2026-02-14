package bench

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Config struct {
	Rows          int
	Lookups       int
	MySQLHost     string
	MySQLPort     int
	MySQLUser     string
	MySQLPassword string
	MySQLDB       string
	PGHost        string
	PGPort        int
	PGUser        string
	PGPassword    string
	PGDB          string
}

type Result struct {
	DB               string
	Table            string
	InsertRows       int
	InsertSeconds    float64
	PointLookupCount int
	PointSeconds     float64
	RangeSeconds     float64
}

func DefaultConfig() Config {
	return Config{
		Rows:          100000,
		Lookups:       20000,
		MySQLHost:     "127.0.0.1",
		MySQLPort:     3306,
		MySQLUser:     "bench",
		MySQLPassword: "bench",
		MySQLDB:       "idbench",
		PGHost:        "127.0.0.1",
		PGPort:        5432,
		PGUser:        "bench",
		PGPassword:    "bench",
		PGDB:          "idbench",
	}
}

func RegisterFlags(fs *flag.FlagSet, cfg *Config) {
	fs.IntVar(&cfg.Rows, "rows", cfg.Rows, "Number of rows to insert for each table.")
	fs.IntVar(&cfg.Lookups, "lookups", cfg.Lookups, "Number of point lookups by primary key.")
	fs.StringVar(&cfg.MySQLHost, "mysql-host", cfg.MySQLHost, "MySQL host")
	fs.IntVar(&cfg.MySQLPort, "mysql-port", cfg.MySQLPort, "MySQL port")
	fs.StringVar(&cfg.MySQLUser, "mysql-user", cfg.MySQLUser, "MySQL user")
	fs.StringVar(&cfg.MySQLPassword, "mysql-password", cfg.MySQLPassword, "MySQL password")
	fs.StringVar(&cfg.MySQLDB, "mysql-db", cfg.MySQLDB, "MySQL database")
	fs.StringVar(&cfg.PGHost, "pg-host", cfg.PGHost, "PostgreSQL host")
	fs.IntVar(&cfg.PGPort, "pg-port", cfg.PGPort, "PostgreSQL port")
	fs.StringVar(&cfg.PGUser, "pg-user", cfg.PGUser, "PostgreSQL user")
	fs.StringVar(&cfg.PGPassword, "pg-password", cfg.PGPassword, "PostgreSQL password")
	fs.StringVar(&cfg.PGDB, "pg-db", cfg.PGDB, "PostgreSQL database")
}

func ValidateConfig(cfg Config) error {
	if cfg.Rows <= 0 {
		return errors.New("rows must be > 0")
	}
	if cfg.Lookups <= 0 {
		return errors.New("lookups must be > 0")
	}
	return nil
}

func UUIDToBytes(u uuid.UUID) []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

func BytesToUUID(b []byte) (uuid.UUID, error) {
	if len(b) != 16 {
		return uuid.Nil, fmt.Errorf("uuid bytes length must be 16, got %d", len(b))
	}
	var u uuid.UUID
	copy(u[:], b)
	return u, nil
}

func FormatResults(results []Result) string {
	var out bytes.Buffer
	out.WriteString("=== Benchmark Results ===\n")
	out.WriteString("db,table,insert_rows,insert_sec,point_lookups,point_sec,range_or_orderby_sec\n")
	for _, r := range results {
		out.WriteString(fmt.Sprintf(
			"%s,%s,%d,%.6f,%d,%.6f,%.6f\n",
			r.DB,
			r.Table,
			r.InsertRows,
			r.InsertSeconds,
			r.PointLookupCount,
			r.PointSeconds,
			r.RangeSeconds,
		))
	}
	return strings.TrimSuffix(out.String(), "\n")
}

func ChunkBounds(total, chunk int) [][2]int {
	if total <= 0 || chunk <= 0 {
		return nil
	}
	out := make([][2]int, 0, (total+chunk-1)/chunk)
	for i := 0; i < total; i += chunk {
		end := i + chunk
		if end > total {
			end = total
		}
		out = append(out, [2]int{i, end})
	}
	return out
}
