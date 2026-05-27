package prober

import (
	"context"
	"fmt"
	"net/http"
	"time"

	ttcrypto "github.com/FsocietyVoid/TrustTrace/internal/crypto"
	pb "github.com/FsocietyVoid/TrustTrace/proto/metrics"
	"github.com/google/uuid"
)

// Prober executes a single probe against a Target and returns a signed ProbeResult.
type Prober struct {
	client  *http.Client
	keyPair *ttcrypto.NodeKeyPair
	region  string
}

// NewProber creates a Prober with the given key pair, region, and HTTP timeout.
func NewProber(kp *ttcrypto.NodeKeyPair, region string, timeout time.Duration) *Prober {
	return &Prober{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("stopped after 3 redirects")
				}
				return nil
			},
		},
		keyPair: kp,
		region:  region,
	}
}

// Probe executes the HTTP check and returns a signed ProbeResult proto.
func (p *Prober) Probe(ctx context.Context, t Target) (*pb.ProbeResult, error) {
	method := t.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, t.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := p.client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	tsNs := start.UnixNano()
	var statusCode int32
	isUp := false

	if err == nil {
		resp.Body.Close()
		statusCode = int32(resp.StatusCode)
		expected := t.ExpectCode
		if expected == 0 {
			expected = http.StatusOK
		}
		isUp = resp.StatusCode == expected
	}

	sig, err := p.keyPair.SignProbe(
		p.keyPair.NodeID, t.URL, tsNs, statusCode, elapsed, isUp,
	)
	if err != nil {
		return nil, fmt.Errorf("sign probe: %w", err)
	}

	return &pb.ProbeResult{
		ProbeId:       uuid.New().String(),
		Region:        p.region,
		TargetUrl:     t.URL,
		TimestampUnix: tsNs,
		StatusCode:    statusCode,
		LatencyMs:     elapsed,
		IsUp:          isUp,
		Signature:     sig,
		PublicKey:     p.keyPair.PublicKey,
		NodeId:        p.keyPair.NodeID,
	}, nil
}
