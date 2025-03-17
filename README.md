# Go Ethereum JSON RPC Proxy

`eth-proxy` proxies the [eth_getBalance](https://besu.hyperledger.org/stable/public-networks/reference/api#eth_getbalance) RPC endpoint from the Ethereum
execution layer.

It supports multiple node clients and implements a failover mechanism

## Running with Docker


Build the image
```
docker build -t eth-proxy . 
```

Run the image with configured node clients

```
docker run -p 8080:8080 -e CLIENT_URLS="https://mainnet.infura.io/v3/ef391c6c612f48f88cae26bc256487be,https://eth-mainnet.g.alchemy.com/v2/HS9PD42pZUxfytFSB2dLRPB5kwr2AePq" eth-proxy
```

## Running tests

```
go test ./...
```