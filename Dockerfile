FROM golang:1.20 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY api/v1/ .

RUN go build -o lunardb-api main.go

FROM alpine:latest

COPY --from=builder /app/lunardb-api /lunardb-api

EXPOSE 8080

CMD {"/lunardb-api"}