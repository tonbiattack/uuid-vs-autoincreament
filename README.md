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

## 複数回実行（平均・標準偏差の自動集計）

`test.bash` は同じ条件を複数回実行し、最後にテーブルごとの平均と標準偏差を出力します。

```bash
# デフォルト: RUNS=3, ROWS=50000, LOOKUPS=10000
bash test.bash

# 件数を変える場合
RUNS=5 ROWS=100000 LOOKUPS=20000 bash test.bash
```

## 実測結果（2026-02-14, ローカル 3回実行）

実行コマンド:

```bash
bash test.bash
```

Run 1:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 83.964825 | 1.303064 | 0.002664 |
| mysql | bench_uuid_char | 92.125588 | 1.330282 | 0.001541 |
| mysql | bench_uuid_bin | 98.474073 | 1.389012 | 0.001330 |
| postgres | bench_auto | 31.615653 | 1.321442 | 0.003991 |
| postgres | bench_uuid | 31.296708 | 1.348341 | 0.003948 |

Run 2:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 95.043433 | 1.275153 | 0.002058 |
| mysql | bench_uuid_char | 101.944038 | 1.344462 | 0.001379 |
| mysql | bench_uuid_bin | 91.580579 | 1.352261 | 0.001173 |
| postgres | bench_auto | 31.406576 | 1.273715 | 0.002223 |
| postgres | bench_uuid | 39.264365 | 1.343136 | 0.002196 |

Run 3:

| db | table | insert_sec | point_sec | range_or_orderby_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 92.660692 | 1.349106 | 0.002082 |
| mysql | bench_uuid_char | 87.747096 | 1.382199 | 0.001335 |
| mysql | bench_uuid_bin | 79.805614 | 1.466573 | 0.002122 |
| postgres | bench_auto | 28.494012 | 1.158750 | 0.001622 |
| postgres | bench_uuid | 28.316125 | 1.329655 | 0.003773 |

平均（3回）:

| db | table | insert_avg_sec | point_avg_sec | range_or_orderby_avg_sec |
| --- | --- | ---: | ---: | ---: |
| mysql | bench_auto | 90.556317 | 1.309108 | 0.002268 |
| mysql | bench_uuid_char | 93.938907 | 1.352314 | 0.001418 |
| mysql | bench_uuid_bin | 89.953422 | 1.402615 | 0.001542 |
| postgres | bench_auto | 30.505414 | 1.251302 | 0.002612 |
| postgres | bench_uuid | 32.959066 | 1.340377 | 0.003306 |

## 評価（上記3回の結果ベース）

- Point Lookup は MySQL/PostgreSQL ともに `bench_auto` が最速。
- Insert は PostgreSQL では `bench_auto` が有利。MySQL では `bench_uuid_bin` と `bench_auto` が僅差。
- MySQL では `bench_uuid_char` より `bench_uuid_bin` の Insert が速い。
- `range_or_orderby_sec` は絶対値が小さく、環境ノイズの影響を受けやすい。

## まとめ（計測結果の解釈）

- 主キーの完全一致検索性能を重視するなら、両DBとも `AUTO_INCREMENT / BIGSERIAL` が第一候補。
- UUID を使うなら、少なくとも MySQL では `CHAR(36)` より `BINARY(16)` を優先する価値が高い。
- PostgreSQL では今回条件で `bench_auto` が Insert / Point Lookup ともに優勢だったため、性能最優先なら連番主キーが無難。
- MySQL の Insert は `bench_auto` と `bench_uuid_bin` が近く、ワークロード次第で逆転しうるため、最終判断は本番に近い件数・同時実行数で再計測する。
- `range_or_orderby_sec` は差が小さいため、この指標単体で主キー方式を決めない。

- さらに精度を上げる場合は `RUNS=5` 以上での比較を推奨。
