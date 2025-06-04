# Go HTTP Load Balancer

This project implements a simple HTTP reverse proxy with a **k+ balancing** strategy.  
Incoming requests are forwarded to one of the configured backends. For every request, `k` backends are randomly sampled and the backend with the fewest active connections is chosen. This approach is similar to the "power of k" load balancing technique and helps distribute load evenly across all servers.

## Building

```
go build ./cmd/app
```

## Running

By default the server listens on `:8080` and proxies requests to two local backends (`http://localhost:8081` and `http://localhost:8082`). You can specify a comma-separated list of backend URLs and the value of `k`:

```
./app -addr :9090 -backends http://srv1:8081,http://srv2:8082,http://srv3:8083 -k 3
```

Run a few HTTP servers on the listed ports to test the load balancer.
