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
SHOW CLUSTER SETTING sql.txn.read_committed_isolation.enabled;

SET CLUSTER SETTING sql.txn.read_committed_isolation.enabled = 'true';

SHOW CLUSTER SETTING sql.txn.read_committed_isolation.enabled;
```

Run workload (SERIALIZABLE)

``` sh
go run load.go \
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

go run load.go \
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

go run load.go \
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

go run load.go \
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

go run load.go \
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
go run load.go \
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

go run load.go \
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

go run load.go \
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

go run load.go \
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

go run load.go \
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

Destroy

``` sh
docker compose down
```
