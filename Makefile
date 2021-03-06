postgres:
	docker run --name postgres12 -p 5432:5432 --network bank-network -e POSTGRES_USER=root -e POSTGRES_PASSWORD=like -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root bank

dropdb:
	docker exec -it postgres12 dropdb bank

migrateup:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up

migrateup1:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose down

migratedown1:
	migrate -path db/migration -database "postgresql://root:like@localhost:5432/bank?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 sqlc test server
