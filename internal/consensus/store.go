package consensus

import (
    "context"
    "fmt"
    "time"

    ttstore "github.com/trusttrace/trusttrace/internal/storage"
    pb "github.com/trusttrace/trusttrace/proto/metrics"
    "go.uber.org/zap"
)

// StoreWorker drains the verified-metric channel and writes to ClickHouse.
type StoreWorker struct {
    ch   <-chan *pb.VerifiedMetric
    ch_c *ttstore.ClickHouseClient
    log  *zap.Logger
}

// NewStoreWorker creates a worker that writes verified metrics to the DB.
func NewStoreWorker(
    ch <-chan *pb.VerifiedMetric,
    clickhouse *ttstore.ClickHouseClient,
    log *zap.Logger,
) *StoreWorker {
    return &StoreWorker{ch: ch, ch_c: clickhouse, log: log}
}

// Run processes verified metrics until ctx is cancelled.
func (sw *StoreWorker) Run(ctx context.Context) {
    batch := make([]*pb.VerifiedMetric, 0, 256)
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    flush := func() {
        if len(batch) == 0 {
            return
        }
        if err := sw.ch_c.InsertMetrics(ctx, batch); err != nil {
            sw.log.Error("clickhouse insert failed", zap.Error(err), zap.Int("count", len(batch)))
        } else {
            sw.log.Debug("flushed batch", zap.Int("count", len(batch)))
        }
        batch = batch[:0]
    }

    for {
        select {
        case <-ctx.Done():
            flush()
            return
        case vm := <-sw.ch:
            batch = append(batch, vm)
            if len(batch) >= 256 {
                flush()
            }
        case <-ticker.C:
            flush()
        }
    }
}

// VerifiedMetricToRow serialises a VerifiedMetric for debug/audit logging.
func VerifiedMetricToRow(vm *pb.VerifiedMetric) string {
    return fmt.Sprintf("[%s] %s → %d | %dms | up=%v | quorum=%d | regions=%v",
        time.Unix(0, vm.TimestampUnix).UTC().Format(time.RFC3339),
        vm.TargetUrl,
        vm.StatusCode,
        vm.LatencyMs,
        vm.IsUp,
        vm.QuorumCount,
        vm.Regions,
    )
}