package notary

import (
	"context"
	"fmt"
	"time"

	ttcrypto "github.com/FsocietyVoid/TrustTrace/internal/crypto"
	"github.com/FsocietyVoid/TrustTrace/internal/storage"
	"go.uber.org/zap"
)

// WindowSize is the fixed 10-minute anchoring window.
const WindowSize = 10 * time.Minute

// Batcher runs on a 10-minute tick, builds a Merkle tree from
// all ClickHouse data in that window, and calls the Anchor+IPFS pipeline.
type Batcher struct {
	store  *storage.ClickHouseClient
	anchor *EthAnchor
	ipfs   *IPFSStore
	log    *zap.Logger
}

// NewBatcher creates a Batcher with all dependencies injected.
func NewBatcher(
	store *storage.ClickHouseClient,
	anchor *EthAnchor,
	ipfs *IPFSStore,
	log *zap.Logger,
) *Batcher {
	return &Batcher{store: store, anchor: anchor, ipfs: ipfs, log: log}
}

// Run processes windows on a fixed tick until ctx is cancelled.
func (b *Batcher) Run(ctx context.Context) {
	// Align to the next clean 10-minute boundary.
	now := time.Now().UTC()
	nextWindow := now.Truncate(WindowSize).Add(WindowSize)
	timer := time.NewTimer(time.Until(nextWindow))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	ticker := time.NewTicker(WindowSize)
	defer ticker.Stop()

	for {
		windowEnd := time.Now().UTC().Truncate(WindowSize)
		windowStart := windowEnd.Add(-WindowSize)
		if err := b.processWindow(ctx, windowStart, windowEnd); err != nil {
			b.log.Error("window processing failed",
				zap.Time("start", windowStart),
				zap.Time("end", windowEnd),
				zap.Error(err),
			)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// processWindow is the core pipeline: query → Merkle → IPFS → blockchain.
func (b *Batcher) processWindow(ctx context.Context, start, end time.Time) error {
	b.log.Info("processing notary window",
		zap.Time("start", start),
		zap.Time("end", end),
	)

	leaves, err := b.store.QueryWindow(ctx, start, end)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	if len(leaves) == 0 {
		b.log.Info("no data in window, skipping anchor")
		return nil
	}

	// Build Merkle tree.
	tree, err := ttcrypto.NewMerkleTree(leaves)
	if err != nil {
		return fmt.Errorf("merkle: %w", err)
	}
	b.log.Info("merkle tree built",
		zap.String("root", tree.RootHex()),
		zap.Int("leaves", len(leaves)),
	)

	// Upload proof blob to IPFS.
	proof := buildProofBlob(tree, start, end, leaves)
	cid, err := b.ipfs.Pin(ctx, proof)
	if err != nil {
		b.log.Warn("IPFS pin failed, continuing without CID", zap.Error(err))
		cid = ""
	}

	// Anchor root hash on-chain.
	txHash, err := b.anchor.Commit(ctx, tree.RootBytes(), start, end)
	if err != nil {
		return fmt.Errorf("blockchain anchor: %w", err)
	}

	b.log.Info("window anchored",
		zap.String("merkle_root", tree.RootHex()),
		zap.String("tx_hash", txHash),
		zap.String("ipfs_cid", cid),
		zap.Int("leaf_count", len(leaves)),
	)
	return nil
}

// buildProofBlob serialises the full Merkle proof for IPFS archival.
func buildProofBlob(tree *ttcrypto.MerkleTree, start, end time.Time, leaves [][]byte) []byte {
	var out []byte
	out = fmt.Appendf(out, "TrustTrace Merkle Proof\n")
	out = fmt.Appendf(out, "Window: %s → %s\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
	out = fmt.Appendf(out, "Root: %s\n", tree.RootHex())
	out = fmt.Appendf(out, "LeafCount: %d\n\n", len(leaves))
	for i, leaf := range leaves {
		proof, _ := tree.Proof(i)
		out = fmt.Appendf(out, "Leaf[%d]: %s\n", i, string(leaf))
		for j, p := range proof {
			out = fmt.Appendf(out, "  Proof[%d]: %x\n", j, p)
		}
	}
	return out
}
