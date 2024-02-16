# iso-load

Run workloads against a Postgres-compatible database with different isolation levels to see how they perform.

### Installation

1. Download the latest version of the iso-load executable from the [Release](https://github.com/codingconcepts/iso-load/releases/latest) page.

2. Extract the executable with the following command:

``` sh
tar -xvf iso-load_VERSION_PLATFORM.tar.gz
```

3. Add the executable to your PATH

4. Run the executable as follows:

``` sh
iso-load --help
Usage of ./iso-load:
  -accounts int
        number of accounts to simulate (default 100000)
  -concurrency int
        number of workers to run concurrently (default 8)
  -duration duration
        duration of test (default 10s)
  -isolation string
        isolation to use [read committed | serializable] (default "serializable")
  -qps int
        number of queries to run per second (default 100)
  -selection int
        number of accounts to work with (default 10)
  -url string
        database connection string (default "postgres://root@localhost:26257?sslmode=disable")
  -version
        show the application version number
```

### Create

I'm spinning up a local Docker cluster but will work against any cluster.

``` sh
docker compose -f compose.yaml up -d

docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure

dw "postgres://root@localhost:26257?sslmode=disable"
enterprise --url "postgres://root@localhost:26257?sslmode=disable"
```

Enable READ COMMITTED

``` sql
SET CLUSTER SETTING sql.txn.read_committed_isolation.enabled = 'true';
```

Run workload (SERIALIZABLE)

``` sh
iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "serializable" \
  --accounts 100000 \
  --selection 100000 \
  --duration 10s \
  --qps 1000

# avg latency:       13ms
# total requests:    6030
# exp total balance: 1000000000
# act total balance: 1000000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "serializable" \
  --accounts 100000 \
  --selection 10000 \
  --duration 10s \
  --qps 1000

# avg latency:       9ms
# total requests:    8849
# exp total balance: 100000000
# act total balance: 100000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "serializable" \
  --accounts 100000 \
  --selection 1000 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9683
# exp total balance: 10000000
# act total balance: 10000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "serializable" \
  --accounts 100000 \
  --selection 100 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9868
# exp total balance: 1000000
# act total balance: 1000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "serializable" \
  --accounts 100000 \
  --selection 10 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9800
# exp total balance: 100000
# act total balance: 100000
```

Run workload (READ COMMITTED)

``` sh
iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "read committed" \
  --accounts 100000 \
  --selection 100000 \
  --duration 10s \
  --qps 1000

# avg latency:       20ms
# total requests:    3791
# exp total balance: 1000000000
# act total balance: 1000000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "read committed" \
  --accounts 100000 \
  --selection 10000 \
  --duration 10s \
  --qps 1000

# avg latency:       14ms
# total requests:    5430
# exp total balance: 100000000
# act total balance: 100000000

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "read committed" \
  --accounts 100000 \
  --selection 1000 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9514
# exp total balance: 10000000
# act total balance: 10000005

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "read committed" \
  --accounts 100000 \
  --selection 100 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9854
# exp total balance: 1000000
# act total balance: 1000180

iso-load \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --isolation "read committed" \
  --accounts 100000 \
  --selection 10 \
  --duration 10s \
  --qps 1000

# avg latency:       8ms
# total requests:    9860
# exp total balance: 100000
# act total balance: 100180
```

### Destroy local infrastructure

``` sh
docker compose down
```
