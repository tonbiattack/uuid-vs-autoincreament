# uuid-vs-autoincreament

MySQL / PostgreSQL で `AUTO_INCREMENT`(連番) と `UUID` 主キーの性能差を測るためのベンチマーク一式です。

## セットアップ

```bash
docker compose up -d
go mod tidy
```

## ベンチマーク実行

```bash
go run ./cmd/benchmark_ids --rows 100000 --lookups 20000
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
