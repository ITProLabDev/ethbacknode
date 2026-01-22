package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"github.com/ITProLabDev/ethbacknode/common/rlp"
	"math/big"
)

// NewEthTxSigner creates a new Ethereum transaction signer with the given parameters.
func NewEthTxSigner(nonce uint64, gasPrice *big.Int, gas uint64, To []byte, value *big.Int, data []byte) *EthTxSigner {
	return &EthTxSigner{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gas,
		To:       &To,
		Value:    value,
		Data:     data,
	}
}

// EthTxSigner represents an Ethereum transaction with signing capabilities.
// Supports EIP-155 chain ID for replay protection.
type EthTxSigner struct {
	chainId  *big.Int
	Nonce    uint64   // nonce of sender account
	GasPrice *big.Int // wei per gas
	Gas      uint64   // gas limit
	To       *[]byte  `rlp:"nil"` // nil means contract creation
	Value    *big.Int // wei amount
	Data     []byte   // contract invocation input data
	V, R, S  *big.Int // signature values
}

// SetChainId sets the EIP-155 chain ID for replay protection.
func (tx *EthTxSigner) SetChainId(id *big.Int) {
	tx.chainId = id
}

// Sign creates an ECDSA signature for the transaction using RFC 6979.
// Returns 65-byte signature in [R || S || V] format and populates V, R, S fields.
func (tx *EthTxSigner) Sign(privateKey *ecdsa.PrivateKey) (sig []byte) {

	digestHash := tx.Hash()
	refId, r, s := SignEcdsaRfc6979(privateKey, digestHash, sha256.New)

	sig = make([]byte, 65)

	//convert r and s to 32 byte slices
	rBytes := padBytes(r.Bytes(), 32)
	sBytes := padBytes(s.Bytes(), 32)
	// Put signature bytes to sig slice
	copy(sig, rBytes)
	copy(sig[32:], sBytes)

	sig[64] = refId
	if tx.chainId == nil {
		tx.chainId = big.NewInt(1)
	}
	chainIdMul := new(big.Int).Mul(tx.chainId, big.NewInt(2))
	v := big.NewInt(int64(refId + 35))
	v.Add(v, chainIdMul)
	//
	tx.V = v
	tx.R = r
	tx.S = s
	//

	return sig
}

// EncodeRPL encodes the signed transaction in RLP format for broadcasting.
func (tx *EthTxSigner) EncodeRPL() (data []byte, err error) {
	rplBuf := new(bytes.Buffer)
	err = rlp.Encode(rplBuf, tx)
	if err != nil {
		return nil, err
	}
	data = rplBuf.Bytes()
	return data, nil
}
// Hash computes the EIP-155 transaction hash for signing.
// Includes chain ID in the hash to prevent replay attacks.
func (tx *EthTxSigner) Hash() []byte {
	return rlpHash([]interface{}{
		tx.Nonce,
		tx.GasPrice,
		tx.Gas,
		tx.To,
		tx.Value,
		tx.Data,
		tx.chainId, uint(0), uint(0),
	})
}
// rlpHash computes Keccak-256 hash of RLP-encoded data.
func rlpHash(x interface{}) []byte {
	rplBuffer := new(bytes.Buffer)
	err := rlp.Encode(rplBuffer, x)
	if err != nil {
		panic(err)
	}
	hash := Keccak256(rplBuffer.Bytes())
	return hash
}
