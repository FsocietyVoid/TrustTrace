package notary

import (
    "bytes"
    "context"
    "fmt"

    shell "github.com/ipfs/go-ipfs-api"
)

// IPFSStore pins proof blobs to IPFS and returns their CID.
type IPFSStore struct {
    sh *shell.Shell
}

// NewIPFSStore creates a client pointing at an IPFS daemon.
func NewIPFSStore(apiURL string) *IPFSStore {
    return &IPFSStore{sh: shell.NewShell(apiURL)}
}

// Pin uploads data to IPFS, pins it, and returns the CID string.
func (s *IPFSStore) Pin(ctx context.Context, data []byte) (string, error) {
    cid, err := s.sh.Add(bytes.NewReader(data), shell.Pin(true))
    if err != nil {
        return "", fmt.Errorf("ipfs add: %w", err)
    }
    return cid, nil
}

// Get retrieves a proof blob from IPFS by CID.
func (s *IPFSStore) Get(ctx context.Context, cid string) ([]byte, error) {
    rc, err := s.sh.Cat(cid)
    if err != nil {
        return nil, fmt.Errorf("ipfs cat: %w", err)
    }
    defer rc.Close()
    var buf bytes.Buffer
    if _, err := buf.ReadFrom(rc); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
