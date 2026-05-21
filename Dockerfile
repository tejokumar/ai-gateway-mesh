FROM golang:1.22 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /out/gateway /app/gateway
COPY configs /app/configs

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
