package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type rpmostree struct {
	*httptest.Server
	response []byte
}

func newRpmostree(response []byte) *rpmostree {
	r := &rpmostree{response: response}
	r.Server = httptest.NewServer(handler(r))
	return r
}

func handler(r *rpmostree) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := w.Write(r.response)
		if err != nil {
			slog.Log(context.Background(), slog.LevelError, "Error writing http response")
		}
	}
}

func expectMetrics(t *testing.T, c prometheus.Collector, fixture string) {
	// #nosec G304
	exp, err := os.Open(path.Join("test", fixture))
	if err != nil {
		t.Fatalf("Error opening fixture file %q: %v", fixture, err)
	}
	if err := testutil.CollectAndCompare(c, exp); err != nil {
		t.Fatal("Unexpected metrics returned:", err)
	}
}

func TestInvalidConfig(t *testing.T) {
	r := newRpmostree([]byte("not,enough,fields"))
	defer r.Close()

	e, _ := NewExporter(*slog.Default())

	expectMetrics(t, e, "invalid_config.metrics")
}
