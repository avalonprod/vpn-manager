FROM golang:1.24.3-alpine3.22 AS builder

WORKDIR /app

COPY . .

RUN go mod download && go get -u ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o ./.bin/app ./cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=0 app/.bin/app .

EXPOSE 8000

CMD ["./app"]