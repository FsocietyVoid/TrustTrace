package telemetry

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus counters/histograms for TrustTrace.
type Metrics struct {
    ProbesTotal       *prometheus.CounterVec
    ProbeLatency      *prometheus.HistogramVec
    QuorumPassTotal   prometheus.Counter
    QuorumFailTotal   prometheus.Counter
    IngestTotal       *prometheus.CounterVec
    MerkleWindowsTotal prometheus.Counter
    AnchorSuccessTotal prometheus.Counter
    AnchorFailTotal    prometheus.Counter
}

// NewMetrics registers and returns all Prometheus metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
    factory := promauto.With(reg)
    return &Metrics{
        ProbesTotal: factory.NewCounterVec(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "prober",
            Name: "probes_total", Help: "Total HTTP probes executed",
        }, []string{"region", "target", "status"}),

        ProbeLatency: factory.NewHistogramVec(prometheus.HistogramOpts{
            Namespace: "trusttrace", Subsystem: "prober",
            Name:    "probe_latency_ms",
            Help:    "HTTP probe round-trip latency in milliseconds",
            Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
        }, []string{"region", "target"}),

        QuorumPassTotal: factory.NewCounter(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "consensus",
            Name: "quorum_pass_total", Help: "Probe windows that reached quorum",
        }),

        QuorumFailTotal: factory.NewCounter(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "consensus",
            Name: "quorum_fail_total", Help: "Probe windows that failed quorum",
        }),

        IngestTotal: factory.NewCounterVec(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "consensus",
            Name: "ingest_total", Help: "Total RPC ingest calls",
        }, []string{"accepted"}),

        MerkleWindowsTotal: factory.NewCounter(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "notary",
            Name: "merkle_windows_total", Help: "10-minute windows processed",
        }),

        AnchorSuccessTotal: factory.NewCounter(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "notary",
            Name: "anchor_success_total", Help: "Successful on-chain anchors",
        }),

        AnchorFailTotal: factory.NewCounter(prometheus.CounterOpts{
            Namespace: "trusttrace", Subsystem: "notary",
            Name: "anchor_fail_total", Help: "Failed on-chain anchors",
        }),
    }
}

// Handler returns an HTTP handler for /metrics.
func Handler() http.Handler {
    return promhttp.Handler()
}
