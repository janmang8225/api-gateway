# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api-gateway ./cmd/gateway

# ---- Run Stage ----
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/api-gateway .

EXPOSE 8080

ENTRYPOINT ["./api-gateway"]
CMD ["--config", "config.yaml"]