code of [Backend master class Golang, Postgres, Docker](https://bit.ly/backendmaster)

---

## Note

#### create database

[dbdiagram.io](https://dbdiagram.io/home)

```
Table accounts as A {
  id bigserial [pk]
  owner varchar [not null]
  balance bigint [not null]
  currency varchar [not null]
  created_at timestamptz [not null, default: `now()`]
  
  Indexes {
    owner
  }
}

Table entries {
  id bigserial [pk]
  account_id bigint [ref: > A.id, not null]
  amount bigint [not null]
  created_at timestamptz [not null, default: `now()`]

  Indexes {
    account_id
  }
}

Table transfers {
  id bigserial [pk]
  from_account_id bigint [ref: > A.id, not null]
  to_account_id bigint [ref: > A.id, not null]
  amount bigint [not null]
  created_at timestamptz [not null, default: `now()`]

  Indexes {
    from_account_id
    to_account_id
    (from_account_id, to_account_id)
  }
}
```

![](http://pp.video.sleen.top/uPic/blog/TZOFxT-bank.png)

#### Install PostgreSQL

```shell
docker pull postgres:12-alpine
```

- Start  

```sh
docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=like -d postgres:12-alpine
```

### datebase  migrate

[golang-migrate](https://github.com/golang-migrate/migrate)

```
cd bank
mkdir -p db/migration
brew install golang-migrate
migrate create -ext sql -dir db/migration -seq init_schema

.
└── db
    └── migration
        ├── 000001_init_schema.down.sql
        └── 000001_init_schema.up.sql
        
```

`vim db/migration/000001_init_schema.down.sql`

```sql
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS accounts;
```

Create a Makefile

```
postgres:
      docker run --name postgres12 -p 5432:5431 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=like -d postgres:12-alpine
	
createdb:
      docker exec -it postgres12 createdb --username=root --owner=root bank

dropdb:
      docker exec -it postgres12 dropdb bank
    
.PHONY: postgres createdb dropdb

```

Migrate db

```
migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose down

migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up
```

Create sql

install [sqlc](https://github.com/kyleconroy/sqlc)

```
sqlc init
```

edit `sqlc.yaml`

```
version: "1"
packages:
  - name: "db"
    path: "./db/sqlc"
    queries: "./db/query/"
    schema: "./db/migration/"
    engine: "postgresql"
    emit_json_tags: true
    emit_prepared_queries: true
    emit_interface: false
    emit_exact_table_names: false
    emit_empty_slices: false
```

 Create a sqlc query file `db/query/account.sql`

```sql
-- name: CreateAccount :one
INSERT INTO accounts (
  owner,
  balance,
  currency
) VALUES (
  $1, $2, $3
)
RETURNING *;
```

generate

```
sqlc generate
```

Generate three files in `db/sqlc/`

```
├── db
│   └── sqlc
│       ├── account.sql.go
│       ├── db.go
│       └── models.go
```

init  `go` `mod`

```
go mod init github.com/xiusl/bank
go mod tidy
```

```
$cat go.mode
module github.com/xiusl/bank

go 1.14
```

add select, delete, update

```
-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: ListAccounts :many
SELECT * FROM accounts
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;

-- name: UpdateAccount :exec
UPDATE accounts
SET balance = $2
WHERE id = $1
RETURNING *;
```







