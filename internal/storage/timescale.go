package storage

import (
	"context"
	"fmt"
	"time"

	pb "github.com/FsocietyVoid/TrustTrace/proto/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TimescaleClient is the fallback write path (TimescaleDB on PostgreSQL).
type TimescaleClient struct {
	pool *pgxpool.Pool
}

// NewTimescaleClient dials TimescaleDB and runs migrations.
func NewTimescaleClient(ctx context.Context, connString string) (*TimescaleClient, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("timescale dial: %w", err)
	}
	c := &TimescaleClient{pool: pool}
	if err := c.migrate(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *TimescaleClient) migrate(ctx context.Context) error {
	_, err := c.pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS verified_metrics (
            probe_id     TEXT        NOT NULL,
            target_url   TEXT        NOT NULL,
            ts           TIMESTAMPTZ NOT NULL,
            status_code  INT         NOT NULL,
            latency_ms   BIGINT      NOT NULL,
            is_up        BOOLEAN     NOT NULL,
            quorum_count INT         NOT NULL,
            regions      TEXT[]
        );
        SELECT create_hypertable('verified_metrics', 'ts', if_not_exists => TRUE);
        CREATE INDEX IF NOT EXISTS idx_vm_url_ts ON verified_metrics (target_url, ts DESC);
    `)
	return err
}

// InsertMetrics upserts a batch of verified metrics.
func (c *TimescaleClient) InsertMetrics(ctx context.Context, metrics []*pb.VerifiedMetric) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, m := range metrics {
		ts := time.Unix(0, m.TimestampUnix)
		_, err := tx.Exec(ctx,
			`INSERT INTO verified_metrics
             (probe_id, target_url, ts, status_code, latency_ms, is_up, quorum_count, regions)
             VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
             ON CONFLICT DO NOTHING`,
			m.ProbeId, m.TargetUrl, ts, m.StatusCode,
			m.LatencyMs, m.IsUp, m.QuorumCount, m.Regions,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
