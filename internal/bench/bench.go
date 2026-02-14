package bench

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Config はベンチマーク実行に必要な件数と接続情報を保持する。
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

// Result は 1 テーブル/1 手法ぶんの計測結果を表す。
type Result struct {
	DB               string
	Table            string
	InsertRows       int
	InsertSeconds    float64
	PointLookupCount int
	PointSeconds     float64
	RangeSeconds     float64
}

// DefaultConfig はローカル実行向けの既定値を返す。
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

// RegisterFlags は Config の各項目を CLI フラグへバインドする。
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

// ValidateConfig は実行前に必須の数値設定を検証する。
func ValidateConfig(cfg Config) error {
	if cfg.Rows <= 0 {
		return errors.New("rows must be > 0")
	}
	if cfg.Lookups <= 0 {
		return errors.New("lookups must be > 0")
	}
	return nil
}

// UUIDToBytes は UUID を 16 バイト配列へコピーして返す。
// DB へ BINARY(16) で保存するための補助関数として使う。
func UUIDToBytes(u uuid.UUID) []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

// BytesToUUID は 16 バイト値を UUID へ復元する。
// 長さが不正な入力は明示的にエラーを返す。
func BytesToUUID(b []byte) (uuid.UUID, error) {
	if len(b) != 16 {
		return uuid.Nil, fmt.Errorf("uuid bytes length must be 16, got %d", len(b))
	}
	var u uuid.UUID
	copy(u[:], b)
	return u, nil
}

// FormatResults は計測結果を見出し付き CSV 文字列に整形する。
func FormatResults(results []Result) string {
	var out bytes.Buffer
	// 先頭に説明行、その次に CSV ヘッダを出力する。
	out.WriteString("=== Benchmark Results ===\n")
	out.WriteString("db,table,insert_rows,insert_sec,point_lookups,point_sec,range_or_orderby_sec\n")
	for _, r := range results {
		// 小数は桁数を固定して比較しやすくする。
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

// ChunkBounds は [start, end) の分割境界を返す。
// 並列処理時に総件数をチャンクへ割り当てる用途を想定する。
func ChunkBounds(total, chunk int) [][2]int {
	if total <= 0 || chunk <= 0 {
		return nil
	}
	// 事前容量を見積もって不要な再確保を減らす。
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
