package consensus

import (
    "crypto/ed25519"
    "sync"
    "time"

    ttcrypto "github.com/trusttrace/trusttrace/internal/crypto"
    pb "github.com/trusttrace/trusttrace/proto/metrics"
)

// QuorumThreshold is the minimum number of agreeing nodes required.
const QuorumThreshold = 2

// bucket accumulates incoming ProbeResults for one (targetURL, 30s-window) key.
type bucket struct {
    mu      sync.Mutex
    results []*pb.ProbeResult
    expiry  time.Time
}

// QuorumManager batches probe results and applies 2-of-3 quorum logic.
type QuorumManager struct {
    mu      sync.Mutex
    buckets map[string]*bucket  // key: targetURL+windowKey
    ttl     time.Duration
    out     chan *pb.VerifiedMetric
}

// NewQuorumManager creates a QuorumManager that emits verified metrics on the returned channel.
func NewQuorumManager(ttl time.Duration) (*QuorumManager, <-chan *pb.VerifiedMetric) {
    ch := make(chan *pb.VerifiedMetric, 512)
    qm := &QuorumManager{
        buckets: make(map[string]*bucket),
        ttl:     ttl,
        out:     ch,
    }
    go qm.reaper()
    return qm, ch
}

// Add attempts to add a result to its bucket and returns a VerifiedMetric if quorum is reached.
func (qm *QuorumManager) Add(r *pb.ProbeResult) {
    // 1. Verify Ed25519 signature.
    if !ttcrypto.VerifyProbe(
        ed25519.PublicKey(r.PublicKey),
        r.NodeId, r.TargetUrl, r.TimestampUnix,
        r.StatusCode, r.LatencyMs, r.IsUp, r.Signature,
    ) {
        return // silently drop forged/invalid probes
    }

    key := windowKey(r.TargetUrl, r.TimestampUnix)

    qm.mu.Lock()
    b, ok := qm.buckets[key]
    if !ok {
        b = &bucket{expiry: time.Now().Add(qm.ttl)}
        qm.buckets[key] = b
    }
    qm.mu.Unlock()

    b.mu.Lock()
    defer b.mu.Unlock()

    // Deduplicate by NodeID within this window.
    for _, existing := range b.results {
        if existing.NodeId == r.NodeId {
            return
        }
    }
    b.results = append(b.results, r)

    if len(b.results) >= QuorumThreshold {
        vm := buildVerifiedMetric(b.results)
        select {
        case qm.out <- vm:
        default:
        }
        // Clear results so quorum fires only once per window.
        b.results = nil
    }
}

func buildVerifiedMetric(results []*pb.ProbeResult) *pb.VerifiedMetric {
    r := results[0]
    regions := make([]string, 0, len(results))
    var totalLatency int64
    for _, res := range results {
        regions = append(regions, res.Region)
        totalLatency += res.LatencyMs
    }
    return &pb.VerifiedMetric{
        ProbeId:       r.ProbeId,
        TargetUrl:     r.TargetUrl,
        TimestampUnix: r.TimestampUnix,
        StatusCode:    r.StatusCode,
        LatencyMs:     totalLatency / int64(len(results)),
        IsUp:          r.IsUp,
        QuorumCount:   int32(len(results)),
        Regions:       regions,
    }
}

// windowKey buckets probe results into 30-second intervals.
func windowKey(url string, tsNs int64) string {
    window := tsNs / int64(30*time.Second)
    return url + ":" + string(rune(window))
}

func (qm *QuorumManager) reaper() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        now := time.Now()
        qm.mu.Lock()
        for k, b := range qm.buckets {
            if now.After(b.expiry) {
                delete(qm.buckets, k)
            }
        }
        qm.mu.Unlock()
    }
}