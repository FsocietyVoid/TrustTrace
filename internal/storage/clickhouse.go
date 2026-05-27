package storage

import (
    "context"
    "fmt"
    "time"

    "github.com/ClickHouse/clickhouse-go/v2"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
    pb "github.com/trusttrace/trusttrace/proto/metrics"
)

// ClickHouseClient wraps the ClickHouse driver.
type ClickHouseClient struct {
    conn driver.Conn
}

// NewClickHouseClient dials ClickHouse and ensures the schema exists.
func NewClickHouseClient(dsn string) (*ClickHouseClient, error) {
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr: []string{dsn},
        Auth: clickhouse.Auth{
            Database: "trusttrace",
            Username: "default",
            Password: "",
        },
        Settings: clickhouse.Settings{
            "max_execution_time": 60,
        },
        DialTimeout:     5 * time.Second,
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: time.Hour,
    })
    if err != nil {
        return nil, fmt.Errorf("clickhouse open: %w", err)
    }
    c := &ClickHouseClient{conn: conn}
    if err := c.migrate(context.Background()); err != nil {
        return nil, fmt.Errorf("clickhouse migrate: %w", err)
    }
    return c, nil
}

// migrate ensures the verified_metrics table exists.
func (c *ClickHouseClient) migrate(ctx context.Context) error {
    ddl := `
    CREATE TABLE IF NOT EXISTS trusttrace.verified_metrics (
        probe_id        String,
        target_url      String,
        timestamp       DateTime64(9, 'UTC'),
        status_code     Int32,
        latency_ms      Int64,
        is_up           Bool,
        quorum_count    Int32,
        regions         Array(String)
    )
    ENGINE = MergeTree()
    PARTITION BY toYYYYMMDD(timestamp)
    ORDER BY (target_url, timestamp)
    TTL timestamp + INTERVAL 90 DAY
    SETTINGS index_granularity = 8192;
    `
    return c.conn.Exec(ctx, ddl)
}

// InsertMetrics batch-inserts a slice of VerifiedMetric records.
func (c *ClickHouseClient) InsertMetrics(ctx context.Context, metrics []*pb.VerifiedMetric) error {
    batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO trusttrace.verified_metrics")
    if err != nil {
        return fmt.Errorf("prepare batch: %w", err)
    }
    for _, m := range metrics {
        ts := time.Unix(0, m.TimestampUnix)
        if err := batch.Append(
            m.ProbeId, m.TargetUrl, ts,
            m.StatusCode, m.LatencyMs, m.IsUp,
            m.QuorumCount, m.Regions,
        ); err != nil {
            return err
        }
    }
    return batch.Send()
}

// QueryWindow retrieves all metrics in a time range for the Merkle batcher.
func (c *ClickHouseClient) QueryWindow(ctx context.Context, start, end time.Time) ([][]byte, error) {
    rows, err := c.conn.Query(ctx,
        `SELECT probe_id, target_url, toUnixTimestamp64Nano(timestamp), status_code, latency_ms, is_up
         FROM trusttrace.verified_metrics
         WHERE timestamp >= $1 AND timestamp < $2
         ORDER BY timestamp ASC`,
        start, end,
    )
    if err != nil {
        return nil, fmt.Errorf("query window: %w", err)
    }
    defer rows.Close()

    var leaves [][]byte
    for rows.Next() {
        var probeID, url string
        var ts, latency int64
        var code int32
        var isUp bool
        if err := rows.Scan(&probeID, &url, &ts, &code, &latency, &isUp); err != nil {
            return nil, err
        }
        // Canonical serialisation for leaf hashing
        leaf := fmt.Appendf(nil, "%s|%s|%d|%d|%d|%v", probeID, url, ts, code, latency, isUp)
        leaves = append(leaves, leaf)
    }
    return leaves, rows.Err()
}