package prober

import (
	"context"
	"time"

	ttcrypto "github.com/FsocietyVoid/TrustTrace/internal/crypto"
	"github.com/FsocietyVoid/TrustTrace/pkg/telemetry"
	pb "github.com/FsocietyVoid/TrustTrace/proto/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// job is a single probe task dispatched to the worker pool.
type job struct {
	target Target
}

// Pool is a fixed-size goroutine pool that probes targets and streams
// signed results to the consensus engine via gRPC.
type Pool struct {
	cfg     Config
	prober  *Prober
	jobs    chan job
	log     *zap.Logger
	client  pb.MetricsIngestionClient
	metrics *telemetry.Metrics
}

// NewPool constructs a Pool, loads/generates the node key, and dials gRPC.
func NewPool(cfg Config, log *zap.Logger, metrics *telemetry.Metrics) (*Pool, error) {
	kp, err := ttcrypto.LoadOrCreateNodeKey(cfg.NodeKeyPath)
	if err != nil {
		return nil, err
	}
	log.Info("edge prober identity", zap.String("node_id", kp.NodeID), zap.String("region", cfg.Region))

	conn, err := grpc.NewClient(cfg.ConsensusAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &Pool{
		cfg:     cfg,
		prober:  NewProber(kp, cfg.Region, cfg.ProbeTimeout),
		jobs:    make(chan job, cfg.QueueDepth),
		log:     log,
		client:  pb.NewMetricsIngestionClient(conn),
		metrics: metrics,
	}, nil
}

// Run starts workers and the scheduler, blocking until ctx is cancelled.
func (p *Pool) Run(ctx context.Context) {
	// Start workers.
	for i := 0; i < p.cfg.WorkerCount; i++ {
		go p.worker(ctx)
	}
	// Schedule targets on a ticker.
	ticker := time.NewTicker(p.cfg.ProbeInterval)
	defer ticker.Stop()
	// Immediate first run.
	p.enqueueAll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.enqueueAll()
		}
	}
}

func (p *Pool) enqueueAll() {
	for _, t := range p.cfg.Targets {
		select {
		case p.jobs <- job{target: t}:
		default:
			p.log.Warn("probe queue full, dropping target", zap.String("url", t.URL))
		}
	}
}

func (p *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-p.jobs:
			p.execute(ctx, j)
		}
	}
}

func (p *Pool) execute(ctx context.Context, j job) {
	result, err := p.prober.Probe(ctx, j.target)
	if err != nil {
		p.log.Error("probe failed", zap.String("url", j.target.URL), zap.Error(err))
		if p.metrics != nil {
			p.metrics.ProbesTotal.With(prometheus.Labels{
				"region": p.cfg.Region,
				"target": j.target.URL,
				"status": "error",
			}).Inc()
		}
		return
	}

	if p.metrics != nil {
		status := "down"
		if result.IsUp {
			status = "up"
		}
		p.metrics.ProbesTotal.With(prometheus.Labels{
			"region": p.cfg.Region,
			"target": j.target.URL,
			"status": status,
		}).Inc()
		p.metrics.ProbeLatency.With(prometheus.Labels{
			"region": p.cfg.Region,
			"target": j.target.URL,
		}).Observe(float64(result.LatencyMs))
	}

	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := p.client.Ingest(ctx2, &pb.IngestRequest{Result: result})
	if err != nil {
		p.log.Error("ingest RPC failed", zap.String("url", j.target.URL), zap.Error(err))
		return
	}
	if !resp.Accepted {
		p.log.Warn("ingest rejected", zap.String("reason", resp.Reason))
	}
}
