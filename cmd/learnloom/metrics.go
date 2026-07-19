package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/VatsalP117/learnloom/internal/execution"
	"github.com/VatsalP117/learnloom/internal/httpapp"
)

func workerMetricsServer(
	address string,
	worker *execution.Worker,
	readiness []httpapp.Readiness,
) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /readyz", func(response http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), 3*time.Second)
		defer cancel()
		for _, dependency := range readiness {
			if err := dependency.Ready(ctx); err != nil {
				response.WriteHeader(http.StatusServiceUnavailable)
				_, _ = response.Write([]byte(`{"status":"unavailable"}`))
				return
			}
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("GET /metrics", func(response http.ResponseWriter, _ *http.Request) {
		snapshot := worker.Snapshot()
		response.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(
			response,
			"# TYPE learnloom_worker_cycles_total counter\nlearnloom_worker_cycles_total %d\n"+
				"# TYPE learnloom_dossiers_generated_total counter\nlearnloom_dossiers_generated_total %d\n"+
				"# TYPE learnloom_dossier_generation_failures_total counter\nlearnloom_dossier_generation_failures_total %d\n"+
				"# TYPE learnloom_deliveries_total counter\nlearnloom_deliveries_total %d\n"+
				"# TYPE learnloom_delivery_failures_total counter\nlearnloom_delivery_failures_total %d\n"+
				"# TYPE learnloom_account_deletions_total counter\nlearnloom_account_deletions_total %d\n"+
				"# TYPE learnloom_worker_last_cycle_timestamp_seconds gauge\nlearnloom_worker_last_cycle_timestamp_seconds %d\n",
			snapshot.Cycles,
			snapshot.Generated,
			snapshot.GenerationFailed,
			snapshot.Delivered,
			snapshot.DeliveryFailed,
			snapshot.Deletions,
			snapshot.LastCycleAt.Unix(),
		)
	})
	return &http.Server{
		Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second,
		IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10,
	}
}
