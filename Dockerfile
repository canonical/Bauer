FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bauer-api ./cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/bauer-api /app/bauer-api

EXPOSE 8090

ENTRYPOINT ["/app/bauer-api"]
