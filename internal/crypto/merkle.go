package crypto

import (
    "crypto/sha256"
    "encoding/hex"
    "errors"
)

// MerkleTree is an in-memory binary Merkle tree over []byte leaves.
type MerkleTree struct {
    Leaves [][]byte
    layers [][][]byte
    Root   []byte
}

// NewMerkleTree constructs a tree from raw leaf data (each leaf is hashed).
func NewMerkleTree(data [][]byte) (*MerkleTree, error) {
    if len(data) == 0 {
        return nil, errors.New("merkle: empty data set")
    }
    leaves := make([][]byte, len(data))
    for i, d := range data {
        h := sha256.Sum256(d)
        leaves[i] = h[:]
    }
    mt := &MerkleTree{Leaves: leaves}
    mt.Root = mt.build(leaves)
    return mt, nil
}

// RootHex returns the hex-encoded root hash.
func (mt *MerkleTree) RootHex() string {
    return hex.EncodeToString(mt.Root)
}

// RootBytes returns the raw 32-byte root hash.
func (mt *MerkleTree) RootBytes() []byte {
    return mt.Root
}

// Proof returns the Merkle proof (sibling hashes) for leaf at index i.
func (mt *MerkleTree) Proof(index int) ([][]byte, error) {
    if index < 0 || index >= len(mt.Leaves) {
        return nil, errors.New("merkle: index out of range")
    }
    var proof [][]byte
    layer := mt.Leaves
    idx := index
    for len(layer) > 1 {
        // pair up; if odd duplicate last
        if len(layer)%2 == 1 {
            layer = append(layer, layer[len(layer)-1])
        }
        if idx%2 == 0 {
            proof = append(proof, layer[idx+1])
        } else {
            proof = append(proof, layer[idx-1])
        }
        // move up
        next := make([][]byte, len(layer)/2)
        for i := 0; i < len(layer); i += 2 {
            combined := append(layer[i], layer[i+1]...)
            h := sha256.Sum256(combined)
            next[i/2] = h[:]
        }
        layer = next
        idx /= 2
    }
    return proof, nil
}

// VerifyProof verifies a Merkle proof for a given leaf and root.
func VerifyProof(leaf []byte, proof [][]byte, root []byte, index int) bool {
    h := sha256.Sum256(leaf)
    current := h[:]
    idx := index
    for _, sibling := range proof {
        var combined []byte
        if idx%2 == 0 {
            combined = append(current, sibling...)
        } else {
            combined = append(sibling, current...)
        }
        h := sha256.Sum256(combined)
        current = h[:]
        idx /= 2
    }
    if len(current) != len(root) {
        return false
    }
    for i := range current {
        if current[i] != root[i] {
            return false
        }
    }
    return true
}

func (mt *MerkleTree) build(layer [][]byte) []byte {
    if len(layer) == 1 {
        return layer[0]
    }
    if len(layer)%2 == 1 {
        layer = append(layer, layer[len(layer)-1])
    }
    next := make([][]byte, len(layer)/2)
    for i := 0; i < len(layer); i += 2 {
        combined := append(layer[i], layer[i+1]...)
        h := sha256.Sum256(combined)
        next[i/2] = h[:]
    }
    mt.layers = append(mt.layers, layer)
    return mt.build(next)
}
