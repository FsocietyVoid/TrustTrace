package prober

import "time"

// Config holds all tunable parameters for one edge prober instance.
type Config struct {
    Region        string        `yaml:"region"`         // e.g. "us-east-1"
    NodeKeyPath   string        `yaml:"node_key_path"`  // path to Ed25519 private key file
    ConsensusAddr string        `yaml:"consensus_addr"` // host:port of gRPC consensus engine
    WorkerCount   int           `yaml:"worker_count"`   // parallel probe goroutines
    QueueDepth    int           `yaml:"queue_depth"`    // buffered channel size
    ProbeInterval time.Duration `yaml:"probe_interval"` // how often to re-probe each target
    ProbeTimeout  time.Duration `yaml:"probe_timeout"`  // per-request HTTP timeout
    Targets       []Target      `yaml:"targets"`
}

// Target describes one HTTP/S endpoint to be monitored.
type Target struct {
    ID         string            `yaml:"id"`
    URL        string            `yaml:"url"`
    Method     string            `yaml:"method"`      // GET | HEAD | POST
    Headers    map[string]string `yaml:"headers"`
    ExpectCode int               `yaml:"expect_code"` // 200 by default
    SLAPercent float64           `yaml:"sla_percent"` // e.g. 99.9
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
    return Config{
        WorkerCount:   20,
        QueueDepth:    500,
        ProbeInterval: 30 * time.Second,
        ProbeTimeout:  5 * time.Second,
    }
}
