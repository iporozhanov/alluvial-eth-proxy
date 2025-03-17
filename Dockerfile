FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o eth-proxy ./cmd/eth-proxy

FROM alpine as certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=builder /app/eth-proxy /eth-proxy
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run the binary
CMD ["/eth-proxy"]