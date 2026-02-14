package bench

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func Testデフォルト設定_既定値を返す(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Rows != 100000 {
		t.Fatalf("Rows = %d, want 100000", cfg.Rows)
	}
	if cfg.Lookups != 20000 {
		t.Fatalf("Lookups = %d, want 20000", cfg.Lookups)
	}
	if cfg.MySQLPort != 3306 {
		t.Fatalf("MySQLPort = %d, want 3306", cfg.MySQLPort)
	}
	if cfg.PGPort != 5432 {
		t.Fatalf("PGPort = %d, want 5432", cfg.PGPort)
	}
}

func TestUUID変換_往復で同一値になる(t *testing.T) {
	u := uuid.New()
	b := UUIDToBytes(u)
	if len(b) != 16 {
		t.Fatalf("len(UUIDToBytes) = %d, want 16", len(b))
	}
	got, err := BytesToUUID(b)
	if err != nil {
		t.Fatalf("BytesToUUID error: %v", err)
	}
	if got != u {
		t.Fatalf("roundtrip mismatch: got %s want %s", got, u)
	}
}

func Test結果整形_CSV形式で出力する(t *testing.T) {
	out := FormatResults([]Result{
		{
			DB:               "mysql",
			Table:            "bench_auto",
			InsertRows:       1000,
			InsertSeconds:    1.23,
			PointLookupCount: 500,
			PointSeconds:     0.45,
			RangeSeconds:     0.01,
		},
	})
	if !strings.Contains(out, "db,table,insert_rows,insert_sec,point_lookups,point_sec,range_or_orderby_sec") {
		t.Fatalf("missing csv header")
	}
	if !strings.Contains(out, "mysql,bench_auto,1000,1.230000,500,0.450000,0.010000") {
		t.Fatalf("missing csv row: %s", out)
	}
}

func Testチャンク境界_分割範囲を返す(t *testing.T) {
	bounds := ChunkBounds(10, 4)
	want := [][2]int{{0, 4}, {4, 8}, {8, 10}}
	if len(bounds) != len(want) {
		t.Fatalf("len = %d, want %d", len(bounds), len(want))
	}
	for i := range bounds {
		if bounds[i] != want[i] {
			t.Fatalf("bounds[%d] = %v, want %v", i, bounds[i], want[i])
		}
	}
}
