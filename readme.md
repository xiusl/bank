code of [Backend master class Golang, Postgres, Docker](https://bit.ly/backendmaster)

---

## Note

#### create database

[dbdiagram.io](https://dbdiagram.io/home)

```yaml
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

```shell
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

```makefile
postgres:
      docker run --name postgres12 -p 5432:5431 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=like -d postgres:12-alpine
	
createdb:
      docker exec -it postgres12 createdb --username=root --owner=root bank

dropdb:
      docker exec -it postgres12 dropdb bank
    
.PHONY: postgres createdb dropdb

```

Migrate db

```shell
migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose down

migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up
```

Create sql

install [sqlc](https://github.com/kyleconroy/sqlc)

```shell
sqlc init
```

edit `sqlc.yaml`

```yaml
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

```shell
sqlc generate
```

Generate three files in `db/sqlc/`

```shell
├── db
│   └── sqlc
│       ├── account.sql.go
│       ├── db.go
│       └── models.go
```

init  `go` `mod`

```shell
go mod init github.com/xiusl/bank
go mod tidy
```

```
$cat go.mode
module github.com/xiusl/bank

go 1.14
```

add select, delete, update

```sql
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

```go
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

```go
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

### TransferTx

create `store.go` file

```go
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := q.tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rb error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

//...
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// update accounts' banlance

		return err
	})

	return result, err
}
```

text transfer tx

```go
ad
```



Finish transfer tx account update

```go
account1, err := q.GetAccount(ctx, arg.FromAccountID)
		if err != nil {
			return err
		}

		result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
			ID:      account1.ID,
			Balance: account1.Balance - arg.Amount,
		})
		if err != nil {
			return err
		}

		account2, err := q.GetAccount(ctx, arg.ToAccountID)
		if err != nil {
			return err
		}

		result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
			ID:      account2.ID,
			Balance: account2.Balance + arg.Amount,
		})
		if err != nil {
			return err
		}
```

Test fail？

account balance error

Update `query/account.sql`

```sql
-- name: GetAccountForUpdate :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1
FOR UPDATE;
```

replace `GetAccount` in  `func (store *Store) TransferTx(...)` 

run test，`FAIL`

```
Error Trace:	store_test.go:38
Error:      	Received unexpected error:
                pq: deadlock detected
Test:       	TestTransferTx
```

`DEADLOCK` 

### Handle deadlock in Golang

```sql
-- name: GetAccountForUpdate :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1
FOR NO KEY UPDATE;
```

refactoring

```sql
-- name: AddAccountBalance :one
UPDATE accounts
SET balance = balance + sqlc.arg(amount)
WHERE id = sqlc.arg(id)
RETURNING *;
```

```go
result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
    ID:     arg.FromAccountID,
    Amount: -arg.Amount,
})
if err != nil {
    return err
}
```



### How to avoid deadlock

To learn

https://www.youtube.com/watch?v=qn3-5wdOfoA&list=PLy_6D98if3ULEtXtNSY_2qN21VCKgoQAE&index=8

### Understand isolation levels & read phenomena in MySQL & PostgreSQL via examples

To learn

 https://www.youtube.com/watch?v=4EajrPgJAk0&list=PLy_6D98if3ULEtXtNSY_2qN21VCKgoQAE&index=9

### Setup Github Actions for Golang + Postgres to run automated tests

`Github` CI

```yaml
// ci.yml
name: ci-test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:

  build:
    name: Build
    runs-on: ubuntu-latest


    services:
      # Label used to access the service container
      postgres:
        # Docker Hub image
        image: postgres:12
        # Provide the password for postgres
        env:
          POSTGRES_USER: root
          POSTGRES_PASSWORD: like
          POSTGRES_DB: bank
        # Set health checks to wait until postgres has started
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          # Maps tcp port 5432 on service container to the host
          - 5432:5432
    steps:

    - name: Set up Go 1.x
      uses: actions/setup-go@v2
      with:
        go-version: ^1.13

    - name: Check out code into the Go module directory
      uses: actions/checkout@v2

    - name: Install Go Migrate
      run: |
        curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
        sudo mv migrate.linux-amd64 /usr/bin/migrate
        which migrate

    - name: Run Migrations
      run: make migrateup

    - name: Test
      run: make test 
```



![20210325154759](http://pp.video.sleen.top/uPic/blog/20210325154759-DT9JIi.jpg)



### Implement RESTful HTTP API in Go using Gin

安装 gin，https://github.com/gin-gonic/gin#installation

```shell
go get -u github.com/gin-gonic/gin
```

![20210325155618](http://pp.video.sleen.top/uPic/blog/20210325155618-zNdeJc.jpg)

项目目录下新建 `api` 目录，添加 `server.go` 文件

```go
package api

import (
    "github.com/gin-gonic/gin"
    db "github.com/xiusl/bank/db/sqlc"
)

// Server http 服务
type Server struct {
    store  *db.Store
    router *gin.Engine
}

// NewServer 创建一个新的服务，并设置路由
func NewServer(store *db.Store) *Server {
    server := &Server{
        store: store,
    }
    router := gin.Default()

    // 设置路由
    router.POST("/accounts", server.createAccount)
    router.GET("/accounts/:id", server.getAccount)
    router.GET("/accounts", server.listAccount)

    server.router = router

    return server
}

// Start 开启服务器，address 监听的地址
func (server *Server) Start(address string) error {
    return server.router.Run(address)
}

// 格式化错误信息
func errorResponse(err error) gin.H {
    return gin.H{"error": err.Error()}
}
```

在 `api` 目录下创建 `account.go` ，增加处理函数

```go
package api

import (
    "database/sql"
    "net/http"

    "github.com/gin-gonic/gin"
    db "github.com/xiusl/bank/db/sqlc"
)

type CreateAccountRequest struct {
    Owner    string `json:"owner" binding:"required"`
    Currency string `json:"currency" binding:"required,oneof=USD EUR"`
}

func (server *Server) createAccount(ctx *gin.Context) {
    var req CreateAccountRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }
    arg := db.CreateAccountParams{
        Owner:    req.Owner,
        Currency: req.Currency,
        Balance:  0,
    }

    account, err := server.store.CreateAccount(ctx, arg)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    ctx.JSON(http.StatusOK, account)
}

type getAccountRequest struct {
    ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getAccount(ctx *gin.Context) {
    var req getAccountRequest
    if err := ctx.ShouldBindUri(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }
    account, err := server.store.GetAccount(ctx, req.ID)
    if err != nil {
        if err == sql.ErrNoRows {
            ctx.JSON(http.StatusNotFound, errorResponse(err))
            return
        }
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    ctx.JSON(http.StatusOK, account)
}

type listAccountRequest struct {
    PageID   int32 `form:"page_id" binding:"required,min=1"`
    PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

func (server *Server) listAccount(ctx *gin.Context) {
    var req listAccountRequest
    if err := ctx.ShouldBindQuery(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    arg := db.ListAccountsParams{
        Limit:  req.PageSize,
        Offset: (req.PageID - 1) * req.PageSize,
    }
    accounts, err := server.store.ListAccounts(ctx, arg)

    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }
    ctx.JSON(http.StatusOK, accounts)
}
```

curl 测试

```shell
$ curl "http://127.0.0.1:8086/accounts?page_id=1&page_size=5"
[{"id":1,"owner":"icmztz","balance":213,"currency":"USD","created_at":"2021-02-24T03:34:11.847889Z"},{"id":2,"owner":"mzlotu","balance":181,"currency":"EUR","created_at":"2021-02-24T03:34:11.850679Z"},{"id":3,"owner":"qvidcd","balance":10,"currency":"EUR","created_at":"2021-02-24T03:34:11.853828Z"},{"id":5,"owner":"czjyob","balance":101,"currency":"CAD","created_at":"2021-02-24T03:34:11.861671Z"},{"id":6,"owner":"edzeub","balance":316,"currency":"CAD","created_at":"2021-02-24T03:34:11.863113Z"}]
```

```shell
$ curl "http://127.0.0.1:8086/accounts/1"
{"id":1,"owner":"icmztz","balance":213,"currency":"USD","created_at":"2021-02-24T03:34:11.847889Z"}
```

```shell
$ curl -X POST -H "Content-Type:application/json" -d '{"owner":"tom", "currency":"USD"}' "http://127.0.0.1:8086/accounts"
{"id":127,"owner":"tom","balance":0,"currency":"USD","created_at":"2021-03-25T10:55:27.629785Z"}
```

修改 `sqlc.yaml` 保证获取列表数据为空时返回空数组

```yaml
emit_empty_slices: true
```

### Load config from file & environment variables in Golang with Viper

