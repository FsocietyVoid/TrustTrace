package consensus

import (
    "sync"
    "time"
)

// DeduplicatorTTL is how long a probe ID is remembered to prevent replays.
const DeduplicatorTTL = 5 * time.Minute

// Deduplicator is a time-bounded bloom-like set for probe IDs.
type Deduplicator struct {
    mu   sync.Mutex
    seen map[string]time.Time
}

// NewDeduplicator returns a Deduplicator with background eviction.
func NewDeduplicator() *Deduplicator {
    d := &Deduplicator{seen: make(map[string]time.Time)}
    go d.evict()
    return d
}

// SeenOrAdd returns true if probeID was already seen; otherwise marks it and returns false.
func (d *Deduplicator) SeenOrAdd(probeID string) bool {
    d.mu.Lock()
    defer d.mu.Unlock()
    if _, ok := d.seen[probeID]; ok {
        return true
    }
    d.seen[probeID] = time.Now().Add(DeduplicatorTTL)
    return false
}

func (d *Deduplicator) evict() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        now := time.Now()
        d.mu.Lock()
        for id, exp := range d.seen {
            if now.After(exp) {
                delete(d.seen, id)
            }
        }
        d.mu.Unlock()
    }
}