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

### unit test

Create `db/sqlc/main_test.go`

```
package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:like@localhost:5432/bank?sslmode=disable"
)

var testQueries *Queries

func TestMain(m *testing.M) {
	conn, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(conn)

	os.Exit(m.Run())

}

```

Need install pq, testify

```
go get github.com/lib/pq
go get github.com/stretchr/testify
```

Create file `db/sqlc/account_test.go`

```
package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	arg := CreateAccountParams{
		Owner:    "tom",
		Balance:  100,
		Currency: "USD",
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)
}

```

Random data 

Create a `util` finder and `random.go` file

```go
package util
const alphabet = "abcdefgijklmnopqrstuvwxyz"
func init() {}
func RandomInt(min, max int64) int64 {}
func RandomString(n int) string {}
func RandomOwner() string {}
func RandomMoney() int64 {}
func RandomCurrency() string {}
```

Finish Account test

```go
package db

createRandomAccount(t *testing.T) Account {}
func TestCreateAccount(t *testing.T) {}
func TestGetAccount(t *testing.T) {}
func TestDeleteAccount(t *testing.T) {}
func TestListAccounts(t *testing.T) {}
```

Add entry and transfer test

```go
package db

func createRandomEntry(t *testing.T, account Account) Entry {}
func TestCreateEntry(t *testing.T) {}
func TestGetEntry(t *testing.T) {}
func TestListEntries(t *testing.T) {}
```

```go
package db

func createRandomTransfer(t *testing.T, account1, account2 Account) Transfer {}
func TestCreateTransfer(t *testing.T) {}
func TestGetTransfer(t *testing.T) {}
func TestListTransfers(t *testing.T) {}
```







