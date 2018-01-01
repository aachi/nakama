# Nakama

## Instructions

Install [CockroachDB](https://www.cockroachlabs.com/), [Git](https://git-scm.com/) and [Go](https://golang.org/).
Then install the _go_ dependencies:
```bash
go get -u github.com/lib/pq
go get -u github.com/go-chi/chi
go get -u github.com/dgrijalva/jwt-go
go get -u github.com/cockroachdb/cockroach-go/crdb
```

Start the database and create the schema:
```bash
cockroach start --insecure --host 127.0.0.1
cat schema.sql | cockroach sql --insecure --format "pretty"
```

Build and run:
```
go build
./nakama
```

`main.go` contains the route definitions; check those.
