# ---------- Build Stage ----------
FROM golang:1.24-alpine3.21 AS builder

WORKDIR /app

# copy mod files & download deps
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o order-service-sample .

# ---------- Runtime Stage ----------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root

# copy binary from builder
COPY --from=builder /app/order-service-sample .

# expose port (gunakan 8080 karena di main.go listen :8080)
EXPOSE 8080

# run the app
CMD ["./order-service-sample"]
