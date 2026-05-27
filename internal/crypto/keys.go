package crypto

import (
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"
)

// NodeKeyPair holds the Ed25519 identity of an edge prober node.
type NodeKeyPair struct {
    PublicKey  ed25519.PublicKey
    PrivateKey ed25519.PrivateKey
    NodeID     string // hex-encoded first 8 bytes of pubkey
}

// GenerateNodeKey creates a fresh Ed25519 key pair.
func GenerateNodeKey() (*NodeKeyPair, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, fmt.Errorf("keygen: %w", err)
    }
    return &NodeKeyPair{
        PublicKey:  pub,
        PrivateKey: priv,
        NodeID:     hex.EncodeToString(pub[:8]),
    }, nil
}

// LoadOrCreateNodeKey loads a key from disk or generates one if absent.
func LoadOrCreateNodeKey(path string) (*NodeKeyPair, error) {
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return nil, err
    }
    data, err := os.ReadFile(path)
    if err == nil && len(data) == ed25519.PrivateKeySize {
        priv := ed25519.PrivateKey(data)
        pub := priv.Public().(ed25519.PublicKey)
        return &NodeKeyPair{
            PublicKey:  pub,
            PrivateKey: priv,
            NodeID:     hex.EncodeToString(pub[:8]),
        }, nil
    }
    kp, err := GenerateNodeKey()
    if err != nil {
        return nil, err
    }
    if err := os.WriteFile(path, kp.PrivateKey, 0600); err != nil {
        return nil, fmt.Errorf("write key: %w", err)
    }
    return kp, nil
}

// SignProbe produces an Ed25519 signature over the canonical probe payload:
//   sha256(nodeID || targetURL || timestampNs || statusCode || latencyMs || isUp)
func (kp *NodeKeyPair) SignProbe(
    nodeID, targetURL string,
    ts int64, statusCode int32, latencyMs int64, isUp bool,
) ([]byte, error) {
    digest := probeDigest(nodeID, targetURL, ts, statusCode, latencyMs, isUp)
    sig := ed25519.Sign(kp.PrivateKey, digest[:])
    return sig, nil
}

// VerifyProbe checks a probe signature against the provided public key.
func VerifyProbe(
    pubKey ed25519.PublicKey,
    nodeID, targetURL string,
    ts int64, statusCode int32, latencyMs int64, isUp bool,
    sig []byte,
) bool {
    digest := probeDigest(nodeID, targetURL, ts, statusCode, latencyMs, isUp)
    return ed25519.Verify(pubKey, digest[:], sig)
}

func probeDigest(
    nodeID, targetURL string,
    ts int64, statusCode int32, latencyMs int64, isUp bool,
) [32]byte {
    h := sha256.New()
    h.Write([]byte(nodeID))
    h.Write([]byte(targetURL))

    var buf [8]byte
    binary.LittleEndian.PutUint64(buf[:], uint64(ts))
    h.Write(buf[:])
    binary.LittleEndian.PutUint32(buf[:4], uint32(statusCode))
    h.Write(buf[:4])
    binary.LittleEndian.PutUint64(buf[:], uint64(latencyMs))
    h.Write(buf[:])
    if isUp {
        h.Write([]byte{1})
    } else {
        h.Write([]byte{0})
    }
    var out [32]byte
    copy(out[:], h.Sum(nil))
    return out
}
