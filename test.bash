for i in 1 2 3; do
  echo "=== run $i ==="
  go run ./cmd/benchmark_ids --rows 100000 --lookups 20000
done
