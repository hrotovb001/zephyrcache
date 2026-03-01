package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ryandielhenn/zephyrcache/internal/telemetry"
	"github.com/ryandielhenn/zephyrcache/pkg/kv"
	"github.com/ryandielhenn/zephyrcache/pkg/node"
	"github.com/ryandielhenn/zephyrcache/pkg/ring"
)

func main() {
	// 1. Initialize this node with routing ring and key value store
	store := kv.NewStore(64 << 20) // 64MB default cap for MVP
	r := ring.New(128, ring.FNV32a)
	id := os.Getenv("SELF_ID")
	addr := os.Getenv("SELF_ADDR")
	seedAddr := os.Getenv("SEED_ADDR")

	r.Add(id, addr)
	n := node.NewNode(store, r, id, node.NormalizeHostPort(addr, "4000"))

	// 2. Run gossip handlers
	go node.StartGossipListener("4000", n)
	if seedAddr != "" {
		n.ConnectToCluster(seedAddr)
	}
	go node.StartGossipPinger(n)

	// 3. Wire up HTTP node endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", n.Healthz)
	mux.HandleFunc("/info", n.Info)
	mux.Handle("/metrics", telemetry.MetricsHandler())
	mux.HandleFunc("/kv/", func(w http.ResponseWriter, req *http.Request) {
		op := methodToOp(req.Method) // "get" | "put" | "post" | "delete" | "other"
		telemetry.Instrument(op, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("Received HTTP Request")
			switch r.Method {
			case http.MethodPut, http.MethodPost:
				n.Put(w, r)
			case http.MethodGet:
				n.Get(w, r)
			case http.MethodDelete:
				n.Del(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, req)
	})

	addr = ":8080"
	fmt.Println("ZephyrCache node listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func methodToOp(m string) string {
	switch m {
	case http.MethodGet:
		return "get"
	case http.MethodPut:
		return "put"
	case http.MethodPost:
		return "post"
	case http.MethodDelete:
		return "delete"
	default:
		return "other"
	}
}
