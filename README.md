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

## 実測結果（2026-02-14, ローカル 2回実行）

実行コマンド:

```bash
go run ./cmd/benchmark_ids --rows 1000 --lookups 2000
```

Run 1:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 2.142952 | 0.391254 | 0.001031 |
| mysql | bench_uuid_char | 2.527369 | 0.398403 | 0.001058 |
| mysql | bench_uuid_bin | 2.323556 | 0.331964 | 0.001039 |
| postgres | bench_auto | 1.019597 | 0.311246 | 0.001042 |
| postgres | bench_uuid | 0.930424 | 0.416015 | 0.001037 |

Run 2:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 2.306792 | 0.335647 | 0.000551 |
| mysql | bench_uuid_char | 2.380259 | 0.410193 | 0.000517 |
| mysql | bench_uuid_bin | 2.264834 | 0.393118 | 0.001040 |
| postgres | bench_auto | 0.877825 | 0.319960 | 0.001027 |
| postgres | bench_uuid | 0.875079 | 0.426984 | 0.001614 |

## 評価（上記2回の結果ベース）

- Insert は PostgreSQL が MySQL より速い傾向（約 0.88-1.02 秒 vs 約 2.14-2.53 秒）。
- MySQL 内比較では、Insert は `bench_auto` が最速、UUID では `bench_uuid_bin` が `bench_uuid_char` より速い傾向。
- Point Lookup は MySQL では `bench_uuid_bin` が良好な回があり、PostgreSQL では `bench_auto` が `bench_uuid` より速い傾向。
- `range_or_orderby_sec` は全体的に非常に短く、差分が小さいためこの条件では優劣を断定しづらい。
- `--lookups 2000` 指定でも `point_lookups` が 1000 なのは、実装が「挿入済み件数（rows）を上限」にしているため。

補足:

- 1000件規模だと揺れが出やすいため、`--rows 100000 --lookups 20000` で複数回実行し、平均・中央値で比較するのがおすすめです。
