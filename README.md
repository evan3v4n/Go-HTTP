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

Run a few HTTP servers on the listed ports to test the load balancer. A quick way
to spin up dummy servers is using Python's built‑in module:

```
python3 -m http.server 8081 &
python3 -m http.server 8082 &
```

With those backends running, start the balancer and issue requests using `curl`:

```
./app -addr :9090 -backends http://localhost:8081,http://localhost:8082 -k 2
curl -v http://localhost:9090
```

Each request will be forwarded to one of the running backends based on the `k+`
algorithm.

## Testing

The repository includes unit tests that validate active connection tracking and
request distribution. To run them along with standard `go vet` checks:

```
go vet ./...
go test ./...
```

Successful output from `go test` looks similar to:

```
?    github.com/evan3v4n/Go-HTTP/cmd/app [no test files]
ok   github.com/evan3v4n/Go-HTTP/internal/logic 0.18s
```

The tests log metrics like `request distribution: [100 101 99]` to help assess
balancing effectiveness.