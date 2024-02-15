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
  --write-percent 50 \
  --duration 1m \
  --qps 1000
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
  --write-percent 50 \
  --duration 1m \
  --qps 1000 \
  --read-committed
```

Destroy

``` sh
docker compose down
```

LOOK FOR GOOD USE CASES FOR READ COMMITTED