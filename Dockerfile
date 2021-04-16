# Build stage
FROM golang:1.16.3-alpine3.13 AS builder
WORKDIR /app
COPY . .
RUN export GO111MODULE=on
RUN export GOPROXY="https://goproxy.io,direct"
RUN go build -o main main.go

# Run stage
FROM alpine:3.13
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .

EXPOSE 8086
CMD ["/app/main"]
