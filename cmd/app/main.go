package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/evan3v4n/Go-HTTP/internal/logic"
)

func main() {
	var backends string
	var addr string
	var k int
	flag.StringVar(&backends, "backends", "http://localhost:8081,http://localhost:8082", "comma separated backend URLs")
	flag.StringVar(&addr, "addr", ":8080", "address to listen on")
	flag.IntVar(&k, "k", 2, "number of backends to sample for k+ balancing")
	flag.Parse()

	urls := strings.Split(backends, ",")
	lb, err := logic.NewLoadBalancer(k, urls)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s with %d backend(s), k=%d", addr, len(urls), k)
	if err := http.ListenAndServe(addr, lb); err != nil {
		log.Fatal(err)
	}
}
