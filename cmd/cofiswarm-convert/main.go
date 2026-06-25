package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keepdevops/cofiswarm-convert/internal/bus"
	"github.com/keepdevops/cofiswarm-convert/internal/httpapi"
	"github.com/keepdevops/cofiswarm-convert/internal/jobs"
	"github.com/keepdevops/cofiswarm-observer-sdk/pkg/buspresence"
	"github.com/keepdevops/cofiswarm-observer-sdk/pkg/servicecomponent"
)

func main() {
	addr := flag.String("listen", ":8015", "listen address")
	flag.Parse()
	q := jobs.New()

	// Optional: announce presence on the observer bus alongside the HTTP API (default-off).
	// COFISWARM_NATS_URL=nats://host:4222 enables it.
	var comp *servicecomponent.Component
	if url := os.Getenv("COFISWARM_NATS_URL"); url != "" {
		nc, cErr := servicecomponent.Connect(url, "cofiswarm-convert")
		if cErr != nil {
			log.Printf("bus connect %s: %v (running without presence)", url, cErr)
		} else {
			defer nc.Close()
			comp = servicecomponent.New(nc, "convert", "convert", bus.Routes())
			if sErr := comp.Start(); sErr != nil {
				log.Printf("bus start: %v (running without presence)", sErr)
				comp = nil
			} else {
				log.Printf("convert announcing presence via %s", url)
			}
		}
	}

	// Carrier presence (broker-free, default-off via COFISWARM_BRIDGE_URL): appear in the
	// observer live roster over the zmq-bridge without needing a NATS broker. HTTP /healthz
	// + /v1/info remain the request/reply surface.
	stopPresence := buspresence.StartPresence(os.Getenv("COFISWARM_BRIDGE_URL"), "convert", map[string]any{"name": "convert"})

	httpSrv := &http.Server{Addr: *addr, Handler: httpapi.New(q).Handler()}
	go func() {
		log.Printf("convert listening on %s", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("convert: server error: %v", err)
		}
	}()

	// On SIGINT/SIGTERM: say goodbye (flip offline now, not after the TTL) then drain HTTP.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Printf("convert: shutting down")
	if comp != nil {
		comp.Shutdown() // NATS goodbye -> offline
	}
	stopPresence() // carrier goodbye -> offline
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		log.Printf("convert: graceful shutdown: %v", err)
	}
}
