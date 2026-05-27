package notary

import (
    "context"
    "crypto/ecdsa"
    "fmt"
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/ethclient"
    "go.uber.org/zap"
)

// anchorMethodID is the first 4 bytes of keccak256("commitRoot(bytes32,uint256,uint256)")
// Pre-computed: 0xa1b2c3d4 (update with actual ABI-encoded selector after deploying contract)
var anchorMethodID = []byte{0xa1, 0xb2, 0xc3, 0xd4}

// EthAnchor submits Merkle root hashes to an Ethereum smart contract.
type EthAnchor struct {
    client          *ethclient.Client
    privKey         *ecdsa.PrivateKey
    contractAddress common.Address
    chainID         *big.Int
    log             *zap.Logger
}

// NewEthAnchor constructs an anchor client connected to an Ethereum RPC.
func NewEthAnchor(rpcURL, privateKeyHex, contractAddr string, chainID int64, log *zap.Logger) (*EthAnchor, error) {
    client, err := ethclient.Dial(rpcURL)
    if err != nil {
        return nil, fmt.Errorf("eth dial: %w", err)
    }
    privKey, err := crypto.HexToECDSA(privateKeyHex)
    if err != nil {
        return nil, fmt.Errorf("parse private key: %w", err)
    }
    return &EthAnchor{
        client:          client,
        privKey:         privKey,
        contractAddress: common.HexToAddress(contractAddr),
        chainID:         big.NewInt(chainID),
        log:             log,
    }, nil
}

// Commit encodes and submits the root hash + window times as a contract call.
// Returns the transaction hash on success.
func (a *EthAnchor) Commit(ctx context.Context, root []byte, windowStart, windowEnd time.Time) (string, error) {
    from := crypto.PubkeyToAddress(a.privKey.PublicKey)

    nonce, err := a.client.PendingNonceAt(ctx, from)
    if err != nil {
        return "", fmt.Errorf("nonce: %w", err)
    }
    gasPrice, err := a.client.SuggestGasPrice(ctx)
    if err != nil {
        return "", fmt.Errorf("gas price: %w", err)
    }

    // ABI-encode: commitRoot(bytes32 root, uint256 windowStart, uint256 windowEnd)
    data := make([]byte, 4+32+32+32)
    copy(data[:4], anchorMethodID)
    copy(data[4:36], padLeft(root, 32))
    copy(data[36:68], padBigInt(big.NewInt(windowStart.Unix()), 32))
    copy(data[68:100], padBigInt(big.NewInt(windowEnd.Unix()), 32))

    tx := types.NewTransaction(
        nonce,
        a.contractAddress,
        big.NewInt(0), // value
        uint64(100_000), // gas limit
        gasPrice,
        data,
    )

    signer := types.NewEIP155Signer(a.chainID)
    signedTx, err := types.SignTx(tx, signer, a.privKey)
    if err != nil {
        return "", fmt.Errorf("sign tx: %w", err)
    }

    if err := a.client.SendTransaction(ctx, signedTx); err != nil {
        return "", fmt.Errorf("send tx: %w", err)
    }

    txHash := signedTx.Hash().Hex()
    a.log.Info("ethereum anchor tx submitted",
        zap.String("tx_hash", txHash),
        zap.String("contract", a.contractAddress.Hex()),
    )
    return txHash, nil
}

// padLeft zero-pads a byte slice to length n (for ABI encoding).
func padLeft(b []byte, n int) []byte {
    if len(b) >= n {
        return b[len(b)-n:]
    }
    padded := make([]byte, n)
    copy(padded[n-len(b):], b)
    return padded
}

func padBigInt(i *big.Int, n int) []byte {
    return padLeft(i.Bytes(), n)
}
