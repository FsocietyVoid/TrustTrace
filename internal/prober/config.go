package prober

import "time"

// Config holds all tunable parameters for one edge prober instance.
type Config struct {
    Region        string        `yaml:"region" mapstructure:"region"`         // e.g. "us-east-1"
    NodeKeyPath   string        `yaml:"node_key_path" mapstructure:"node_key_path"`  // path to Ed25519 private key file
    ConsensusAddr string        `yaml:"consensus_addr" mapstructure:"consensus_addr"` // host:port of gRPC consensus engine
    WorkerCount   int           `yaml:"worker_count" mapstructure:"worker_count"`   // parallel probe goroutines
    QueueDepth    int           `yaml:"queue_depth" mapstructure:"queue_depth"`    // buffered channel size
    ProbeInterval time.Duration `yaml:"probe_interval" mapstructure:"probe_interval"` // how often to re-probe each target
    ProbeTimeout  time.Duration `yaml:"probe_timeout" mapstructure:"probe_timeout"`  // per-request HTTP timeout
    Targets       []Target      `yaml:"targets" mapstructure:"targets"`
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
