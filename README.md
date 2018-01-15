# Nakama

## Instructions

Install [CockroachDB](https://www.cockroachlabs.com/), [Git](https://git-scm.com/) and [Go](https://golang.org/).
Then install the _go_ dependencies:
```bash
go get -u github.com/lib/pq
go get -u github.com/go-chi/chi
go get -u github.com/dgrijalva/jwt-go
go get -u github.com/cockroachdb/cockroach-go/crdb
go get -u github.com/gernest/mention
```

Start the database and create the schema:
```bash
cockroach start --insecure --host 127.0.0.1
cat schema.sql | cockroach sql --insecure
```

Build and run:
```
go build
./nakama
```

`main.go` contains the route definitions; check those.

curl -H "Content-Type: application/json" -X POST -d '{"email":"john@example.dev"}' http://localhost:8081/api/login | jq '.'

curl http://localhost:8081/api/posts/1 -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NDc1NjYzNTAsInN1YiI6IjEifQ.dadQ_QZqMQ03f62hxbo4nkKD_kyJ_SU1C7Md6u9O26o'


