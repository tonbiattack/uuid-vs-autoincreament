#!/usr/bin/env bash
set -euo pipefail

RUNS="${RUNS:-3}"
ROWS="${ROWS:-50000}"
LOOKUPS="${LOOKUPS:-10000}"

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

for i in $(seq 1 "$RUNS"); do
  echo "=== run $i ==="
  out="$(go run ./cmd/benchmark_ids --rows "$ROWS" --lookups "$LOOKUPS")"
  printf '%s\n' "$out"
  printf '%s\n' "$out" | awk -F, '/^(mysql|postgres),/ { print }' >> "$tmp_file"
done

if [[ ! -s "$tmp_file" ]]; then
  echo "no benchmark rows found"
  exit 1
fi

echo "=== Summary (mean ± stddev, sec) ==="
echo "db,table,runs,insert_rows,point_lookups,insert_avg,insert_stddev,point_avg,point_stddev,range_or_orderby_avg,range_or_orderby_stddev"
awk -F, '
{
  key = $1 "," $2
  n[key]++
  insert_rows[key] = $3
  point_lookups[key] = $5

  ins_sum[key] += $4
  ins_sq[key] += $4 * $4

  point_sum[key] += $6
  point_sq[key] += $6 * $6

  range_sum[key] += $7
  range_sq[key] += $7 * $7
}
END {
  for (k in n) {
    count = n[k]
    split(k, parts, ",")

    ins_avg = ins_sum[k] / count
    point_avg = point_sum[k] / count
    range_avg = range_sum[k] / count

    ins_var = (ins_sq[k] / count) - (ins_avg * ins_avg)
    point_var = (point_sq[k] / count) - (point_avg * point_avg)
    range_var = (range_sq[k] / count) - (range_avg * range_avg)

    if (ins_var < 0) ins_var = 0
    if (point_var < 0) point_var = 0
    if (range_var < 0) range_var = 0

    printf "%s,%s,%d,%s,%s,%.6f,%.6f,%.6f,%.6f,%.6f,%.6f\n",
      parts[1], parts[2], count, insert_rows[k], point_lookups[k],
      ins_avg, sqrt(ins_var), point_avg, sqrt(point_var), range_avg, sqrt(range_var)
  }
}
' "$tmp_file" | sort -t, -k1,1 -k2,2
