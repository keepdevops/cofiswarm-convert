package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/keepdevops/cofiswarm-convert/internal/httpapi"
	"github.com/keepdevops/cofiswarm-convert/internal/jobs"
)

func main() {
	addr := flag.String("listen", ":8015", "listen address")
	flag.Parse()
	q := jobs.New()
	log.Printf("convert listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, httpapi.New(q).Handler()))
}
