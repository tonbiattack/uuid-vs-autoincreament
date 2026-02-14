import argparse
import statistics
import time
import uuid
from dataclasses import dataclass
from typing import Iterable, List, Sequence

import mysql.connector
import psycopg


@dataclass
class BenchResult:
    db: str
    table: str
    insert_rows: int
    insert_seconds: float
    point_lookup_count: int
    point_lookup_seconds: float
    range_scan_seconds: float


def chunked(values: Sequence, size: int) -> Iterable[Sequence]:
    for i in range(0, len(values), size):
        yield values[i : i + size]


def mysql_connect(args: argparse.Namespace):
    return mysql.connector.connect(
        host=args.mysql_host,
        port=args.mysql_port,
        user=args.mysql_user,
        password=args.mysql_password,
        database=args.mysql_db,
        autocommit=False,
    )


def pg_connect(args: argparse.Namespace):
    return psycopg.connect(
        host=args.pg_host,
        port=args.pg_port,
        user=args.pg_user,
        password=args.pg_password,
        dbname=args.pg_db,
        autocommit=False,
    )


def setup_mysql(conn):
    cur = conn.cursor()
    cur.execute("DROP TABLE IF EXISTS bench_auto")
    cur.execute("DROP TABLE IF EXISTS bench_uuid_char")
    cur.execute("DROP TABLE IF EXISTS bench_uuid_bin")

    cur.execute(
        """
        CREATE TABLE bench_auto (
            id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
            payload VARCHAR(100) NOT NULL
        ) ENGINE=InnoDB
        """
    )
    cur.execute(
        """
        CREATE TABLE bench_uuid_char (
            id CHAR(36) NOT NULL PRIMARY KEY,
            payload VARCHAR(100) NOT NULL
        ) ENGINE=InnoDB
        """
    )
    cur.execute(
        """
        CREATE TABLE bench_uuid_bin (
            id BINARY(16) NOT NULL PRIMARY KEY,
            payload VARCHAR(100) NOT NULL
        ) ENGINE=InnoDB
        """
    )
    conn.commit()
    cur.close()


def setup_postgres(conn):
    with conn.cursor() as cur:
        cur.execute("DROP TABLE IF EXISTS bench_auto")
        cur.execute("DROP TABLE IF EXISTS bench_uuid")
        cur.execute(
            """
            CREATE TABLE bench_auto (
                id BIGSERIAL PRIMARY KEY,
                payload TEXT NOT NULL
            )
            """
        )
        cur.execute(
            """
            CREATE TABLE bench_uuid (
                id UUID PRIMARY KEY,
                payload TEXT NOT NULL
            )
            """
        )
    conn.commit()


def bench_mysql_auto(conn, rows: int, lookups: int) -> BenchResult:
    payloads = [f"p-{i}" for i in range(rows)]
    cur = conn.cursor()

    start = time.perf_counter()
    for block in chunked(payloads, 2000):
        cur.executemany(
            "INSERT INTO bench_auto (payload) VALUES (%s)",
            [(x,) for x in block],
        )
    conn.commit()
    insert_sec = time.perf_counter() - start

    cur.execute("SELECT id FROM bench_auto ORDER BY id")
    ids = [r[0] for r in cur.fetchall()]
    sample_ids = ids[:lookups] if lookups <= len(ids) else ids

    start = time.perf_counter()
    for v in sample_ids:
        cur.execute("SELECT payload FROM bench_auto WHERE id = %s", (v,))
        cur.fetchone()
    point_sec = time.perf_counter() - start

    if ids:
        lo = ids[len(ids) // 4]
        hi = ids[(len(ids) * 3) // 4]
    else:
        lo = 0
        hi = 0

    start = time.perf_counter()
    cur.execute("SELECT COUNT(*) FROM bench_auto WHERE id BETWEEN %s AND %s", (lo, hi))
    cur.fetchone()
    range_sec = time.perf_counter() - start

    cur.close()
    return BenchResult(
        db="mysql",
        table="bench_auto",
        insert_rows=rows,
        insert_seconds=insert_sec,
        point_lookup_count=len(sample_ids),
        point_lookup_seconds=point_sec,
        range_scan_seconds=range_sec,
    )


def bench_mysql_uuid_char(conn, rows: int, lookups: int) -> BenchResult:
    ids = [str(uuid.uuid4()) for _ in range(rows)]
    pairs = [(ids[i], f"p-{i}") for i in range(rows)]
    cur = conn.cursor()

    start = time.perf_counter()
    for block in chunked(pairs, 1000):
        cur.executemany(
            "INSERT INTO bench_uuid_char (id, payload) VALUES (%s, %s)",
            block,
        )
    conn.commit()
    insert_sec = time.perf_counter() - start

    sample_ids = ids[:lookups] if lookups <= len(ids) else ids
    start = time.perf_counter()
    for v in sample_ids:
        cur.execute("SELECT payload FROM bench_uuid_char WHERE id = %s", (v,))
        cur.fetchone()
    point_sec = time.perf_counter() - start

    start = time.perf_counter()
    cur.execute("SELECT id FROM bench_uuid_char ORDER BY id LIMIT 10000")
    cur.fetchall()
    range_sec = time.perf_counter() - start

    cur.close()
    return BenchResult(
        db="mysql",
        table="bench_uuid_char",
        insert_rows=rows,
        insert_seconds=insert_sec,
        point_lookup_count=len(sample_ids),
        point_lookup_seconds=point_sec,
        range_scan_seconds=range_sec,
    )


def bench_mysql_uuid_bin(conn, rows: int, lookups: int) -> BenchResult:
    ids = [uuid.uuid4().bytes for _ in range(rows)]
    pairs = [(ids[i], f"p-{i}") for i in range(rows)]
    cur = conn.cursor()

    start = time.perf_counter()
    for block in chunked(pairs, 1000):
        cur.executemany(
            "INSERT INTO bench_uuid_bin (id, payload) VALUES (%s, %s)",
            block,
        )
    conn.commit()
    insert_sec = time.perf_counter() - start

    sample_ids = ids[:lookups] if lookups <= len(ids) else ids
    start = time.perf_counter()
    for v in sample_ids:
        cur.execute("SELECT payload FROM bench_uuid_bin WHERE id = %s", (v,))
        cur.fetchone()
    point_sec = time.perf_counter() - start

    start = time.perf_counter()
    cur.execute("SELECT id FROM bench_uuid_bin ORDER BY id LIMIT 10000")
    cur.fetchall()
    range_sec = time.perf_counter() - start

    cur.close()
    return BenchResult(
        db="mysql",
        table="bench_uuid_bin",
        insert_rows=rows,
        insert_seconds=insert_sec,
        point_lookup_count=len(sample_ids),
        point_lookup_seconds=point_sec,
        range_scan_seconds=range_sec,
    )


def bench_pg_auto(conn, rows: int, lookups: int) -> BenchResult:
    payloads = [(f"p-{i}",) for i in range(rows)]
    with conn.cursor() as cur:
        start = time.perf_counter()
        for block in chunked(payloads, 2000):
            cur.executemany("INSERT INTO bench_auto (payload) VALUES (%s)", block)
        conn.commit()
        insert_sec = time.perf_counter() - start

        cur.execute("SELECT id FROM bench_auto ORDER BY id")
        ids = [r[0] for r in cur.fetchall()]
        sample_ids = ids[:lookups] if lookups <= len(ids) else ids

        start = time.perf_counter()
        for v in sample_ids:
            cur.execute("SELECT payload FROM bench_auto WHERE id = %s", (v,))
            cur.fetchone()
        point_sec = time.perf_counter() - start

        if ids:
            lo = ids[len(ids) // 4]
            hi = ids[(len(ids) * 3) // 4]
        else:
            lo = 0
            hi = 0
        start = time.perf_counter()
        cur.execute("SELECT COUNT(*) FROM bench_auto WHERE id BETWEEN %s AND %s", (lo, hi))
        cur.fetchone()
        range_sec = time.perf_counter() - start

    return BenchResult(
        db="postgres",
        table="bench_auto",
        insert_rows=rows,
        insert_seconds=insert_sec,
        point_lookup_count=len(sample_ids),
        point_lookup_seconds=point_sec,
        range_scan_seconds=range_sec,
    )


def bench_pg_uuid(conn, rows: int, lookups: int) -> BenchResult:
    ids = [uuid.uuid4() for _ in range(rows)]
    pairs = [(ids[i], f"p-{i}") for i in range(rows)]
    with conn.cursor() as cur:
        start = time.perf_counter()
        for block in chunked(pairs, 1000):
            cur.executemany("INSERT INTO bench_uuid (id, payload) VALUES (%s, %s)", block)
        conn.commit()
        insert_sec = time.perf_counter() - start

        sample_ids = ids[:lookups] if lookups <= len(ids) else ids
        start = time.perf_counter()
        for v in sample_ids:
            cur.execute("SELECT payload FROM bench_uuid WHERE id = %s", (v,))
            cur.fetchone()
        point_sec = time.perf_counter() - start

        start = time.perf_counter()
        cur.execute("SELECT id FROM bench_uuid ORDER BY id LIMIT 10000")
        cur.fetchall()
        range_sec = time.perf_counter() - start

    return BenchResult(
        db="postgres",
        table="bench_uuid",
        insert_rows=rows,
        insert_seconds=insert_sec,
        point_lookup_count=len(sample_ids),
        point_lookup_seconds=point_sec,
        range_scan_seconds=range_sec,
    )


def print_results(results: List[BenchResult]):
    print("\n=== Benchmark Results ===")
    print("db,table,insert_rows,insert_sec,point_lookups,point_sec,range_or_orderby_sec")
    for r in results:
        print(
            f"{r.db},{r.table},{r.insert_rows},{r.insert_seconds:.6f},"
            f"{r.point_lookup_count},{r.point_lookup_seconds:.6f},{r.range_scan_seconds:.6f}"
        )

    insert_by_db = {}
    for r in results:
        insert_by_db.setdefault(r.db, []).append(r.insert_seconds)
    for db, vals in insert_by_db.items():
        print(f"{db}: insert mean={statistics.mean(vals):.6f}s")


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Benchmark AUTO_INCREMENT vs UUID on MySQL and PostgreSQL.")
    p.add_argument("--rows", type=int, default=100_000, help="Number of rows to insert for each table.")
    p.add_argument("--lookups", type=int, default=20_000, help="Number of point lookups by primary key.")

    p.add_argument("--mysql-host", default="127.0.0.1")
    p.add_argument("--mysql-port", type=int, default=3306)
    p.add_argument("--mysql-user", default="bench")
    p.add_argument("--mysql-password", default="bench")
    p.add_argument("--mysql-db", default="idbench")

    p.add_argument("--pg-host", default="127.0.0.1")
    p.add_argument("--pg-port", type=int, default=5432)
    p.add_argument("--pg-user", default="bench")
    p.add_argument("--pg-password", default="bench")
    p.add_argument("--pg-db", default="idbench")
    return p.parse_args()


def main():
    args = parse_args()
    all_results: List[BenchResult] = []

    mysql_conn = mysql_connect(args)
    try:
        setup_mysql(mysql_conn)
        all_results.append(bench_mysql_auto(mysql_conn, args.rows, args.lookups))
        all_results.append(bench_mysql_uuid_char(mysql_conn, args.rows, args.lookups))
        all_results.append(bench_mysql_uuid_bin(mysql_conn, args.rows, args.lookups))
    finally:
        mysql_conn.close()

    pg_conn = pg_connect(args)
    try:
        setup_postgres(pg_conn)
        all_results.append(bench_pg_auto(pg_conn, args.rows, args.lookups))
        all_results.append(bench_pg_uuid(pg_conn, args.rows, args.lookups))
    finally:
        pg_conn.close()

    print_results(all_results)


if __name__ == "__main__":
    main()
