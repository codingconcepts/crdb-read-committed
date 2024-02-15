### Create

I'm spinning up a local Docker cluster but will work against any cluster.

``` sh
docker compose -f compose.yaml up -d

docker exec -it node1 cockroach init --insecure
docker exec -it node1 cockroach sql --insecure

enterprise --url "postgres://root@localhost:26257?sslmode=disable"
```

Init workload

``` sh
go run load.go init \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --seed-rows 10
```

Run workload (SERIALIZABLE)

``` sh
go run load.go run \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --write-percent 10 \
  --duration 10s \
  --qps 100
```

Enable READ COMMITTED

``` sql
SHOW CLUSTER SETTING sql.txn.read_committed_isolation.enabled;

SET CLUSTER SETTING sql.txn.read_committed_isolation.enabled = 'true';

SHOW CLUSTER SETTING sql.txn.read_committed_isolation.enabled;
```

Run workload (READ COMMITTED)

``` sh
go run load.go run \
  --url "postgres://root@localhost:26257?sslmode=disable" \
  --write-percent 10 \
  --duration 10s \
  --qps 1000 \
  --read-committed
```

Destroy

``` sh
docker compose down
```

```
s - 10
avg read latency:  23ms
avg write latency: 89ms

s - 50
avg read latency:  23ms
avg write latency: 85ms

rc - 10
avg read latency:  21ms
avg write latency: 87ms

rc - 50
avg read latency:  23ms
avg write latency: 90ms
```