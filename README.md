# uuid-vs-autoincreament

MySQL / PostgreSQL で `AUTO_INCREMENT`(連番) と `UUID` 主キーの性能差を測るためのベンチマーク一式です。

## セットアップ

```bash
docker compose up -d
go mod tidy
```

## ベンチマーク実行

```bash
# 1) 軽量確認
go run ./cmd/benchmark_ids --rows 10000 --lookups 2000

# 2) 本番比較（推奨）
go run ./cmd/benchmark_ids --rows 100000 --lookups 20000

# 3) 余裕があれば
go run ./cmd/benchmark_ids --rows 300000 --lookups 60000
```

実行すると以下を計測します。

- Insert: 指定件数の一括挿入
- Point Lookup: 主キー完全一致検索
- Range Scan: 連番主キーの範囲検索 (`UUID` は `ORDER BY + LIMIT` を代替計測)

## 計測対象テーブル

- MySQL
- `bench_auto`: `BIGINT AUTO_INCREMENT`
- `bench_uuid_char`: `CHAR(36)` (UUID文字列)
- `bench_uuid_bin`: `BINARY(16)` (UUIDバイナリ)

- PostgreSQL
- `bench_auto`: `BIGSERIAL`
- `bench_uuid`: `UUID` 型

## オプション

```bash
go run ./cmd/benchmark_ids --help
```

主なオプション:

- `--rows`: 挿入件数
- `--lookups`: 主キー検索回数
- `--mysql-host`, `--mysql-port`, `--mysql-user`, `--mysql-password`
- `--pg-host`, `--pg-port`, `--pg-user`, `--pg-password`

## 実測結果（2026-02-14, ローカル 3回実行）

実行コマンド:

```bash
for i in 1 2 3; do
  echo "=== run $i ==="
  go run ./cmd/benchmark_ids --rows 10000 --lookups 2000
done
```

Run 1:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 18.192566 | 0.271273 | 0.000938 |
| mysql | bench_uuid_char | 18.778375 | 0.272474 | 0.001301 |
| mysql | bench_uuid_bin | 40.900365 | 0.296230 | 0.001380 |
| postgres | bench_auto | 6.069807 | 0.240588 | 0.000842 |
| postgres | bench_uuid | 6.342978 | 0.321057 | 0.003509 |

Run 2:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 18.302677 | 0.287904 | 0.000843 |
| mysql | bench_uuid_char | 18.567982 | 0.286554 | 0.001286 |
| mysql | bench_uuid_bin | 18.016458 | 0.268081 | 0.001124 |
| postgres | bench_auto | 6.214916 | 0.235860 | 0.000850 |
| postgres | bench_uuid | 6.431976 | 0.243786 | 0.003214 |

Run 3:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 18.146893 | 0.276196 | 0.000689 |
| mysql | bench_uuid_char | 18.042769 | 0.293410 | 0.001228 |
| mysql | bench_uuid_bin | 17.877042 | 0.293287 | 0.001147 |
| postgres | bench_auto | 6.170988 | 0.297306 | 0.000847 |
| postgres | bench_uuid | 6.288569 | 0.283096 | 0.003483 |

平均（3回）:

| db | table | avg_insert_sec | avg_point_sec | avg_range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 18.214045 | 0.278458 | 0.000823 |
| mysql | bench_uuid_char | 18.463042 | 0.284146 | 0.001272 |
| mysql | bench_uuid_bin | 25.597955 | 0.285866 | 0.001217 |
| postgres | bench_auto | 6.151904 | 0.257918 | 0.000846 |
| postgres | bench_uuid | 6.354508 | 0.282646 | 0.003402 |

## UUID vs AUTO_INCREMENT（今回データの要約）

- MySQL（平均）
- Insert: `AUTO_INCREMENT` (`bench_auto` 18.214s) が `UUID(CHAR)` (18.463s) と `UUID(BINARY)` (25.598s) より速い。
- Point Lookup: `AUTO_INCREMENT` (0.278s) と UUID 系 (0.284-0.286s) はほぼ同等。

- PostgreSQL（平均）
- Insert: `AUTO_INCREMENT` 相当 (`bench_auto` 6.152s) が `UUID` (6.355s) より速い。
- Point Lookup: `AUTO_INCREMENT` 相当 (0.258s) が `UUID` (0.283s) より速い。

- 結論（この3回の条件）
- 一貫して `AUTO_INCREMENT` 側が有利。特に PostgreSQL の Point Lookup と、MySQL の Insert（vs UUID BINARY）で差が出た。

## 考察（上記3回の結果ベース）

- 全体として PostgreSQL の Insert は MySQL より速い（約 6.15-6.35 秒 vs 約 18.21-25.60 秒）。
- MySQL の Insert は `bench_auto` が最速で、`bench_uuid_char` はわずかに遅い。`bench_uuid_bin` は Run 1 の 40.9 秒が平均を押し上げた。
- MySQL の Point Lookup は `bench_auto` / `bench_uuid_char` / `bench_uuid_bin` がほぼ同水準（0.27-0.29 秒台）。
- PostgreSQL は Insert/Point Lookup ともに `bench_auto` が `bench_uuid` より速い傾向。
- `range_or_orderby_sec` は PostgreSQL の `bench_uuid` が他より一貫して遅い（約 0.0032-0.0035 秒）。

補足:

- MySQL `bench_uuid_bin` の Run 1（40.900365 秒）は外れ値の可能性があるため、中央値でも比較すると傾向を判断しやすいです。
- より安定した比較のため、`--rows 100000 --lookups 20000` で複数回実行し、平均・中央値を併記するのがおすすめです。
