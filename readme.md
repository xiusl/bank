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

安装 `viper` ，https://github.com/spf13/viper

```shell
go get github.com/spf13/viper
```

在项目目录下新增 `app.env` 文件

```yaml
DB_DRIVER=postgres
DB_SOURCE=postgresql://root:loke@localhost:5432/back?sslmode=disable
SERVER_ADDRESS=0.0.0.0:8086
```

在 `util` 目录下增加 `config.go` 用来解析 `app.env`，

```go
package util

import "github.com/spf13/viper"

type Config struct {
    DBDriver      string `mapstructure:"DB_DRIVER"`
    DBSource      string `mapstructure:"DB_SOURCE"`
    ServerAddress string `mapstructure:"SERVER_ADDRESS"`
}

func LoadConfig(path string) (config Config, err error) {
    viper.AddConfigPath(path)
    viper.SetConfigName("app")
    viper.SetConfigType("env")

    viper.AutomaticEnv()

    if err = viper.ReadInConfig(); err != nil {
        return
    }

    err = viper.Unmarshal(&config)
    return
}
```

修改 `main.go`

```go
package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
    "github.com/xiusl/bank/api"
    db "github.com/xiusl/bank/db/sqlc"
    "github.com/xiusl/bank/util"
)

func main() {
    config, err := util.LoadConfig(".")
    if err != nil {
        log.Fatal("cannot load config:", err)
        return
    }
    conn, err := sql.Open(config.DBDriver, config.DBSource)
    if err != nil {
        log.Fatal("cannot connect to db:", err)
    }

    store := db.NewStore(conn)
    server := api.NewServer(store)

    err = server.Start(config.ServerAddress)
    if err != nil {
        log.Fatal("connot start server:", err)
    }
}
```

完善`sqlc/main_test.go` 文件

```go
func TestMain(m *testing.M) {
    config, err:= util.LoadConfig("../..")
    if err != nil {
        log.Fatal("connot load config:", err)
        return
    }

    testDB, err = sql.Open(config.DBDriver, config.DBSource)
    if err != nil {
        log.Fatal("cannot connect to db:", err)
    }

    testQueries = New(testDB)

    os.Exit(m.Run())
}
```

### Mock DB for testing HTTP API in Go and achieve 100% coverage

安装 mock https://github.com/golang/mock

```
go install github.com/golang/mock/mockgen@v1.5.0
```

执行下面命令

```
mockgen -destination db/mock/store.go github.com/xiusl/bank/db/sqlc Store
```

修改 `sql.yaml` ，使用接口形式

```yaml
emit_interface: true
```

重新 `make sqlc`

在 `api` 目录下创建 `main_test.go` 

```go
package api

import (
    "os"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
    gin.SetMode(gin.TestMode)

    os.Exit(m.Run())
}
```

测试 `GetAccount`

```go
// api/account_test.go
package api

import (
    "database/sql"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/require"

    mockdb "github.com/xiusl/bank/db/mock"
    db "github.com/xiusl/bank/db/sqlc"
    "github.com/xiusl/bank/util"
)

func TestGetAccount(t *testing.T) {
    testCases := []struct {
        name         string
        accountID    int64
        buildStuds   func(store *mockdb.MockStore)
        exceptStatus int
    }{
        {
            name:      "OK",
            accountID: 1,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 1
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(id)).
                    Return(randomAccount(id), nil).
                    Times(1)
            },
            exceptStatus: http.StatusOK,
        },
        {
            name:      "Not Found",
            accountID: 2,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 2
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(id)).
                    Return(db.Account{}, sql.ErrNoRows).
                    Times(1)
            },
            exceptStatus: http.StatusNotFound,
        },
        {
            name:      "InternalError",
            accountID: 3,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 3
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(id)).
                    Return(db.Account{}, sql.ErrConnDone).
                    Times(1)
            },
            exceptStatus: http.StatusInternalServerError,
        },
        {
            name:      "BadRequest",
            accountID: 0,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 0
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            exceptStatus: http.StatusBadRequest,
        },
    }

    for i := range testCases {
        tc := testCases[i]
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStuds(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            url := fmt.Sprintf("/accounts/%d", tc.accountID)
            request, err := http.NewRequest(http.MethodGet, url, nil)
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            require.Equal(t, tc.exceptStatus, recorder.Code)
        })
    }
}

func randomAccount(id int64) db.Account {
    return db.Account{
        ID:        id,
        Owner:     util.RandomOwner(),
        Balance:   util.RandomMoney(),
        Currency:  util.RandomCurrency(),
        CreatedAt: time.Now(),
    }
}
```

测试 `make test`

```shell
=== RUN   TestGetAccount
=== RUN   TestGetAccount/OK
[GIN] 2021/03/26 - 10:53:05 | 200 |     359.804µs |                 | GET      "/accounts/1"
=== RUN   TestGetAccount/Not_Found
[GIN] 2021/03/26 - 10:53:05 | 404 |       22.98µs |                 | GET      "/accounts/2"
=== RUN   TestGetAccount/InternalError
[GIN] 2021/03/26 - 10:53:05 | 500 |       29.27µs |                 | GET      "/accounts/3"
=== RUN   TestGetAccount/BadRequest
[GIN] 2021/03/26 - 10:53:05 | 400 |      31.383µs |                 | GET      "/accounts/0"
--- PASS: TestGetAccount (0.00s)
    --- PASS: TestGetAccount/OK (0.00s)
    --- PASS: TestGetAccount/Not_Found (0.00s)
    --- PASS: TestGetAccount/InternalError (0.00s)
    --- PASS: TestGetAccount/BadRequest (0.00s)
PASS
coverage: 48.8% of statements
ok      github.com/xiusl/bank/api       (cached)        coverage: 48.8% of statements
```

#### Mock 的使用

TODO

#### 测试 CreateAccount

```go
package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/require"

    mockdb "github.com/xiusl/bank/db/mock"
    db "github.com/xiusl/bank/db/sqlc"
    "github.com/xiusl/bank/util"
)

func TestGetAccountAPI(t *testing.T) {
    account := randomAccount()
    testCases := []struct {
        name         string
        accountID    int64
        buildStuds   func(store *mockdb.MockStore)
        exceptStatus int
    }{
        {
            name:      "OK",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(account, nil).
                    Times(1)
            },
            exceptStatus: http.StatusOK,
        },
        {
            name:      "Not Found",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(db.Account{}, sql.ErrNoRows).
                    Times(1)
            },
            exceptStatus: http.StatusNotFound,
        },
        {
            name:      "InternalError",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(db.Account{}, sql.ErrConnDone).
                    Times(1)
            },
            exceptStatus: http.StatusInternalServerError,
        },
        {
            name:      "InvalidID",
            accountID: 0,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 0
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            exceptStatus: http.StatusBadRequest,
        },
    }

    for i := range testCases {
        tc := testCases[i]
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStuds(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            url := fmt.Sprintf("/accounts/%d", tc.accountID)
            request, err := http.NewRequest(http.MethodGet, url, nil)
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            require.Equal(t, tc.exceptStatus, recorder.Code)
        })
    }
}

func TestCreateAccountAPI(t *testing.T) {
    account := randomAccount()

    testCases := []struct {
        name        string
        body        gin.H
        buildStuds  func(store *mockdb.MockStore)
        expectSatus int
    }{
        {
            name: "OK",
            body: gin.H{
                "owner":    account.Owner,
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                arg := db.CreateAccountParams{
                    Owner:    account.Owner,
                    Currency: account.Currency,
                    Balance:  0,
                }
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Eq(arg)).
                    Times(1).
                    Return(account, nil)
            },
            expectSatus: http.StatusOK,
        },
        {
            name: "InvalidCurrency",
            body: gin.H{
                "owner":    account.Owner,
                "currency": "invalid",
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            expectSatus: http.StatusBadRequest,
        },
        {
            name: "InvalidOwner",
            body: gin.H{
                "owner":    "",
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            expectSatus: http.StatusBadRequest,
        },
        {
            name: "InternalError",
            body: gin.H{
                "owner":    account.Owner,
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.Account{}, sql.ErrConnDone)
            },
            expectSatus: http.StatusInternalServerError,
        },
    }

    for i := range testCases {
        tc := testCases[i]

        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStuds(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            data, err := json.Marshal(tc.body)
            require.NoError(t, err)

            url := "/accounts"
            request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            require.Equal(t, tc.expectSatus, recorder.Code)

        })
    }
}

func randomAccount() db.Account {
    return db.Account{
        ID:        util.RandomInt(1, 1000),
        Owner:     util.RandomOwner(),
        Balance:   util.RandomMoney(),
        Currency:  util.RandomCurrency(),
        CreatedAt: time.Now(),
    }
}
```

测试

![20210326114702](http://pp.video.sleen.top/uPic/blog/20210326114702-hrgk81.jpg)

#### 检查响应内容

```go
package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/require"

    mockdb "github.com/xiusl/bank/db/mock"
    db "github.com/xiusl/bank/db/sqlc"
    "github.com/xiusl/bank/util"
)

func TestGetAccountAPI(t *testing.T) {
    account := randomAccount()
    testCases := []struct {
        name          string
        accountID     int64
        buildStuds    func(store *mockdb.MockStore)
        checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
    }{
        {
            name:      "OK",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(account, nil).
                    Times(1)
            },
            checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recorder.Code)
                requireBodyMatchAccount(t, recorder.Body, account)
            },
        },
        {
            name:      "Not Found",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(db.Account{}, sql.ErrNoRows).
                    Times(1)
            },
            checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusNotFound, recorder.Code)
            },
        },
        {
            name:      "InternalError",
            accountID: account.ID,
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
                    Return(db.Account{}, sql.ErrConnDone).
                    Times(1)
            },
            checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
        {
            name:      "InvalidID",
            accountID: 0,
            buildStuds: func(store *mockdb.MockStore) {
                const id int64 = 0
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
    }

    for i := range testCases {
        tc := testCases[i]
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStuds(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            url := fmt.Sprintf("/accounts/%d", tc.accountID)
            request, err := http.NewRequest(http.MethodGet, url, nil)
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(t, recorder)
        })
    }
}

// 测试创建账户 API
func TestCreateAccountAPI(t *testing.T) {
    account := randomAccount()

    testCases := []struct {
        name          string
        body          gin.H
        buildStuds    func(store *mockdb.MockStore)
        checkResponse func(recorder *httptest.ResponseRecorder)
    }{
        {
            name: "OK",
            body: gin.H{
                "owner":    account.Owner,
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                arg := db.CreateAccountParams{
                    Owner:    account.Owner,
                    Currency: account.Currency,
                    Balance:  0,
                }
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Eq(arg)).
                    Times(1).
                    Return(account, nil)
            },
            checkResponse: func(recoder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recoder.Code)
                requireBodyMatchAccount(t, recoder.Body, account)
            },
        },
        {
            name: "InvalidCurrency",
            body: gin.H{
                "owner":    account.Owner,
                "currency": "invalid",
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recoder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recoder.Code)
            },
        },
        {
            name: "InvalidOwner",
            body: gin.H{
                "owner":    "",
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recoder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recoder.Code)
            },
        },
        {
            name: "InternalError",
            body: gin.H{
                "owner":    account.Owner,
                "currency": account.Currency,
            },
            buildStuds: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateAccount(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.Account{}, sql.ErrConnDone)
            },
            checkResponse: func(recoder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recoder.Code)
            },
        },
    }

    for i := range testCases {
        tc := testCases[i]

        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStuds(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            data, err := json.Marshal(tc.body)
            require.NoError(t, err)

            url := "/accounts"
            request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(recorder)

        })
    }
}

func randomAccount() db.Account {
    return db.Account{
        ID:       util.RandomInt(1, 1000),
        Owner:    util.RandomOwner(),
        Balance:  util.RandomMoney(),
        Currency: util.RandomCurrency(),
    }
}

func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
    data, err := ioutil.ReadAll(body)
    require.NoError(t, err)

    var gotAccount db.Account
    err = json.Unmarshal(data, &gotAccount)
    require.NoError(t, err)
    require.Equal(t, account, gotAccount)
}
```

#### 优化 makefile

`makefile` 新增 `mock`

```makefile
mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/xiusl/back/db/sqlc Store

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test server mock
```

#### 测试 ListAccount API

```go
func TestListAccountsAPI(t *testing.T) {
    n := 5
    accounts := make([]db.Account, n)
    for i := 0; i < n; i++ {
        accounts[i] = randomAccount()
    }

    type Query struct {
        pageID   int
        pageSize int
    }

    testCases := []struct {
        name          string
        query         Query
        buildStubs    func(store *mockdb.MockStore)
        checkResponse func(recorder *httptest.ResponseRecorder)
    }{
        {
            name: "OK",
            query: Query{
                pageID:   1,
                pageSize: n,
            },
            buildStubs: func(store *mockdb.MockStore) {
                arg := db.ListAccountsParams{
                    Limit:  int32(n),
                    Offset: 0,
                }

                store.EXPECT().
                    ListAccounts(gomock.Any(), gomock.Eq(arg)).
                    Times(1).
                    Return(accounts, nil)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recorder.Code)
                requireBodyMatchAccounts(t, recorder.Body, accounts)
            },
        },
        {
            name: "InternalError",
            query: Query{
                pageID:   1,
                pageSize: n,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    ListAccounts(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return([]db.Account{}, sql.ErrConnDone)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
        {
            name: "InvalidPageID",
            query: Query{
                pageID:   -1,
                pageSize: n,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    ListAccounts(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "InvalidPageSize",
            query: Query{
                pageID:   1,
                pageSize: 1000,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    ListAccounts(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
    }

    for i := range testCases {
        tc := testCases[i]

        t.Run(tc.name, func(t *testing.T) {

            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStubs(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            url := "/accounts"
            request, err := http.NewRequest(http.MethodGet, url, nil)
            require.NoError(t, err)

            q := request.URL.Query()
            q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
            q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
            request.URL.RawQuery = q.Encode()

            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(recorder)

        })
    }
}


func requireBodyMatchAccounts(t *testing.T, body *bytes.Buffer, accouns []db.Account) {
    data, err := ioutil.ReadAll(body)
    require.NoError(t, err)

    var gotAccounts []db.Account
    err = json.Unmarshal(data, &gotAccounts)
    require.NoError(t, err)
    require.Equal(t, gotAccounts, accouns)
}

```

### Transfer

```go
package api

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    db "github.com/xiusl/bank/db/sqlc"
)

type transferRequest struct {
    FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
    ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
    Amount        int64  `json:"amount" binding:"required,min=1"`
    Currency      string `json:"currency" binding:"required,oneof=USD EUR"`
}

func (server *Server) goodAccountCurrency(ctx *gin.Context, accountID int64, currency string) bool {
    account, err := server.store.GetAccount(ctx, accountID)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return false
    }

    if account.Currency != currency {
        err := fmt.Errorf("account [%d] currency mismatch:%s vs %s", accountID, account.Currency, currency)
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return false
    }

    return true
}

func (server *Server) createTransfer(ctx *gin.Context) {
    var req transferRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    if !server.goodAccountCurrency(ctx, req.FromAccountID, req.Currency) {
        return
    }
    if !server.goodAccountCurrency(ctx, req.ToAccountID, req.Currency) {
        return
    }

    arg := db.TransferTxParams{
        FromAccountID: req.FromAccountID,
        ToAccountID:   req.ToAccountID,
        Amount:        req.Amount,
    }
    result, err := server.store.TransferTx(ctx, arg)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return
    }

    ctx.JSON(http.StatusOK, result)
}

```

####  自定义参数验证器

安装 `https://github.com/go-playground/validator`

```shell
go get github.com/go-playground/validator/v10
```

```go
// validator.go
package api

import (
    "github.com/go-playground/validator/v10"
)

var supportedCurrencies = map[string]bool{
    "USD": true,
    "ERU": true,
}

var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
    if currency, ok := fieldLevel.Field().Interface().(string); ok {
        return supportedCurrencies[currency]
    }
    return false
}
```

修改 `server.go`

```go
func NewServer(store db.Store) *Server {
    server := &Server{
        store: store,
    }
    router := gin.Default()

    // 设置路由
    router.POST("/accounts", server.createAccount)
    router.GET("/accounts/:id", server.getAccount)
    router.GET("/accounts", server.listAccount)
    router.POST("/transfers", server.createTransfer)

    // 注册验证器
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("currency", validCurrency)
    }

    server.router = router

    return server
}
```

优化 `transfer.go` 函数名

```
goodAccountCurrency -> sameAccountCurrency
```

### 测试 Transfer

优化转账前账号的验证

```go
func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) bool {
    account, err := server.store.GetAccount(ctx, accountID)
    if err != nil {
        if err == sql.ErrNoRows {
            ctx.JSON(http.StatusNotFound, err)
            return false
        }
        ctx.JSON(http.StatusInternalServerError, errorResponse(err))
        return false
    }

    if account.Currency != currency {
        err := fmt.Errorf("account [%d] currency mismatch:%s vs %s", accountID, account.Currency, currency)
        ctx.JSON(http.StatusBadRequest, errorResponse(err))
        return false
    }

    return true
}
```

编写测试 `transfer_test.go`

```go
package api

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/require"
    mockdb "github.com/xiusl/bank/db/mock"
    db "github.com/xiusl/bank/db/sqlc"
)

func TestTransferAPI(t *testing.T) {
    amount := int64(10)

    account1 := randomAccount()
    account2 := randomAccount()
    account3 := randomAccount()

    account1.Currency = "USD"
    account2.Currency = "USD"
    account3.Currency = "EUR"

    testCases := []struct {
        name          string
        body          gin.H
        buildStubs    func(store *mockdb.MockStore)
        checkResponse func(recorder *httptest.ResponseRecorder)
    }{
        {
            name: "OK",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
                    Times(1).Return(account1, nil)
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
                    Times(1).Return(account2, nil)

                arg := db.TransferTxParams{
                    FromAccountID: account1.ID,
                    ToAccountID:   account2.ID,
                    Amount:        amount,
                }
                store.EXPECT().
                    TransferTx(gomock.Any(), gomock.Eq(arg)).Times(1)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recorder.Code)
            },
        },
        {
            name: "FromAccountNotFound",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
                    Times(1).Return(db.Account{}, sql.ErrNoRows)
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
                    Times(0)
                store.EXPECT().
                    TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusNotFound, recorder.Code)
            },
        },
        {
            name: "ToAccountNotFound",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
                    Times(1).Return(account1, nil)
                store.EXPECT().
                    GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
                    Times(1).Return(db.Account{}, sql.ErrNoRows)
                store.EXPECT().
                    TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusNotFound, recorder.Code)
            },
        },
        {
            name: "FromAccountCurrencyMismatch",
            body: gin.H{
                "from_account_id": account3.ID,
                "to_account_id":   account1.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account3.ID)).Times(1).Return(account3, nil)
                store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(0)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "ToAccountCurrencyMismatch",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account3.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Times(1).Return(account1, nil)
                store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account3.ID)).Times(1).Return(account3, nil)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "InvalidCurrency",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "abc",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "NegativeAmount",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          -amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "GetAccountError",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(1).Return(db.Account{}, sql.ErrConnDone)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
        {
            name: "TransferTxError",
            body: gin.H{
                "from_account_id": account1.ID,
                "to_account_id":   account2.ID,
                "amount":          amount,
                "currency":        "USD",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().GetAccount(gomock.Any(), account1.ID).Times(1).Return(account1, nil)
                store.EXPECT().GetAccount(gomock.Any(), account2.ID).Times(1).Return(account2, nil)
                store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(1).Return(db.TransferTxResult{}, sql.ErrConnDone)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
    }

    for i := range testCases {
        tc := testCases[i]
        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStubs(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            data, err := json.Marshal(tc.body)
            require.NoError(t, err)

            url := "/transfers"
            request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(recorder)

        })
    }

}
```

### 添加 User 表

```sql

Table users as U {
  username varchar [pk]
  hashed_password varchar [not null]
  full_name varchar [not null]
  email varchar [unique, not null]
  password_changed_at timestamptz [not null, default: `0001-01-01 00:00:00Z`]
  created_at timestamptz [not null, default: `now()`]
}

Table accounts as A {
  id bigserial [pk]
  owner varchar [ref: > U.username, not null]
  balance bigint [not null]
  currency varchar [not null]
  created_at timestamptz [not null, default: `now()`]
  
  Indexes {
    owner
    (owner, currency) [unique]
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

![20210329110443](http://pp.video.sleen.top/uPic/blog/20210329110443-LvkbaT.jpg)

PostgreSQL

```sql
CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT ('0001-01-01 00:00:00Z'),
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "accounts" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username");

ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");

CREATE INDEX ON "accounts" ("owner");

CREATE UNIQUE INDEX ON "accounts" ("owner", "currency");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");

```

创建一个新迭代的数据库迁移

```shell
migrate create -ext sql -dir db/migration -seq add_user 

- db 
    - mirgration
        - 000002_add_user.down.sql
        - 000002_add_user.up.sql
```

编辑迁移代码

```sql
// 000002_add_user.down.sql
ALTER TABLE IF EXISTS "accounts" DROP CONSTRAINT IF EXISTS "owner_currency_key";

ALTER TABLE IF EXISTS "accounts" DROP CONSTRAINT IF EXISTS "accounts_owner_fkey";

DROP TABLE IF EXISTS "users";
```

```sql
// 000001_add_user_up.sql
CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar UNIQUE NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT ('0001-01-01 00:00:00Z'),
  "created_at" timestamptz NOT NULL DEFAULT (now())
);


ALTER TABLE "accounts" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username");

// 一个用户只能拥有一种币种的账号
// CREATE UNIQUE INDEX ON "accounts" ("owner", "currency");
ALTER TABLE "accounts" ADD CONSTRAINT "owner_currency_key" UNIQUE ("owner", "currency");

```

进行数据库迁移

```shell
migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up
2021/03/29 11:23:25 error: Dirty database version 2. Fix and force version.
make: *** [migrateup] Error 1
```

出现错误 `Dirty database` ，手动修改 `schema_migrations` 的 `drity` 为 `FALSE`

再次执行 `make migratedown` `make migrateup`

为 `makefile` 新增命令

```makefile
migrateup1:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up 1

migratedown1:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose down 1

.PHONY: migrateup1, migratedown1
```

创建 `query/user.sql` 

```sql
-- name: CreateUser :one
INSERT INTO users (
    username,
    hashed_password,
    full_name,
    email
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;
```

执行 `make sqlc`

为 `user.sql.go`





### User api

