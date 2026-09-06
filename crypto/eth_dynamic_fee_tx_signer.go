package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/ITProLabDev/ethbacknode/common/rlp"
)

// TxTypeDynamicFee is the EIP-2718 type byte of an EIP-1559 transaction.
//
// A typed transaction is not RLP at the top level: it is this byte followed
// by the RLP of its fields, and that whole string is what gets hashed and
// what eth_sendRawTransaction takes. The legacy envelope (EthTxSigner) has no
// such byte, which is how a decoder tells them apart -- a first byte of 0xc0
// or above is an RLP list and so legacy, anything below is a type.
const TxTypeDynamicFee byte = 0x02

var (
	// ErrNoChainID reports signing with no chain id set. Unlike EthTxSigner,
	// which silently assumes mainnet, a dynamic-fee transaction refuses:
	// guessing the chain on a newer, more deliberately-threaded send path is
	// a way to sign for the wrong network by omission.
	ErrNoChainID = errors.New("crypto: chain id is not set")

	// ErrNoFeeCap reports signing with no MaxFeePerGas set. A nil cap would
	// encode as zero, which offers nothing and is not what a caller who left
	// the field alone meant.
	ErrNoFeeCap = errors.New("crypto: maxFeePerGas is not set")

	// ErrNotSigned reports EncodeRPL before Sign.
	ErrNotSigned = errors.New("crypto: transaction is not signed")
)

// AccessTuple is one entry in an EIP-2930 access list: an address and the
// storage slots of it the transaction declares it will touch. Declaring them
// costs gas up front and makes the accesses themselves cheaper. Nothing here
// fills one in automatically -- that is a caller's decision -- but the field
// is part of the envelope and has to be encoded either way.
type AccessTuple struct {
	// Address is the 20-byte account.
	Address []byte
	// StorageKeys are 32 bytes each.
	StorageKeys [][]byte
}

// NewEthDynamicFeeTxSigner builds an unsigned EIP-1559 (type 2) transaction.
// Pass a nil recipient to create a contract.
func NewEthDynamicFeeTxSigner(nonce uint64, maxPriorityFeePerGas, maxFeePerGas *big.Int, gas uint64, to []byte, value *big.Int, data []byte) *EthDynamicFeeTxSigner {
	return &EthDynamicFeeTxSigner{
		Nonce:                nonce,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		MaxFeePerGas:         maxFeePerGas,
		Gas:                  gas,
		To:                   &to,
		Value:                value,
		Data:                 data,
	}
}

// EthDynamicFeeTxSigner is an EIP-1559 (type 2) transaction and the signing
// that turns it into something broadcastable.
//
// It is a separate type from EthTxSigner rather than a shared one because the
// two price a transaction in different currencies of thought. A legacy
// transaction names one price and pays it. A dynamic-fee transaction names
// two figures -- the most it will pay in total, and the most it will pay the
// proposer above the base fee -- and the chain charges the base fee plus the
// tip, refunding the difference. Sharing one struct would mean fee fields
// that are meaningless half the time.
//
// Replay protection needs no EIP-155 folding here: the chain id is a field of
// the payload, so the signature covers it directly and the parity byte stays
// 0 or 1.
type EthDynamicFeeTxSigner struct {
	chainId *big.Int

	Nonce uint64 // sender's transaction count

	// MaxPriorityFeePerGas is the most this transaction will pay the
	// proposer per unit of gas, on top of the base fee. Nil is zero, which
	// is a valid offer on a chain with room in its blocks and an unmineable
	// one otherwise.
	MaxPriorityFeePerGas *big.Int

	// MaxFeePerGas is the ceiling on base fee plus tip. It must be set: this
	// is the figure that says how much the transaction can cost.
	MaxFeePerGas *big.Int

	Gas   uint64   // gas limit
	To    *[]byte  // recipient; nil creates a contract
	Value *big.Int // wei
	Data  []byte   // call data

	// AccessList is the EIP-2930 list. Empty is normal and encodes as an
	// empty list.
	AccessList []AccessTuple

	// YParity is the recovery bit, 0 or 1. It is not a V: folding the chain
	// id into it the way EIP-155 does would make the transaction undecodable.
	YParity, R, S *big.Int
}

// SetChainId sets the chain the transaction is valid on. It must be set
// before SigningHash/Sign.
func (tx *EthDynamicFeeTxSigner) SetChainId(id *big.Int) { tx.chainId = id }

// ChainId returns the chain id this transaction is bound to, or nil.
func (tx *EthDynamicFeeTxSigner) ChainId() *big.Int { return tx.chainId }

// encodeAccessList renders the access list as nested RLP items: a list of
// [address, [storage keys]] pairs.
func encodeAccessList(list []AccessTuple) []interface{} {
	out := make([]interface{}, 0, len(list))
	for _, tuple := range list {
		keys := make([]interface{}, 0, len(tuple.StorageKeys))
		for _, key := range tuple.StorageKeys {
			keys = append(keys, key)
		}
		out = append(out, []interface{}{tuple.Address, keys})
	}
	return out
}

// payload returns the nine fields the signature covers, in order.
func (tx *EthDynamicFeeTxSigner) payload() []interface{} {
	return []interface{}{
		tx.chainId,
		tx.Nonce,
		tx.MaxPriorityFeePerGas,
		tx.MaxFeePerGas,
		tx.Gas,
		tx.To,
		tx.Value,
		tx.Data,
		encodeAccessList(tx.AccessList),
	}
}

// SigningHash is the keccak-256 of the type byte followed by the RLP of the
// nine payload fields.
//
// The type byte is inside the hash, which is what stops a signature made for
// one envelope being replayed in another: the same nine numbers under a
// different type byte hash differently.
func (tx *EthDynamicFeeTxSigner) SigningHash() ([]byte, error) {
	if tx.chainId == nil {
		return nil, ErrNoChainID
	}
	if tx.MaxFeePerGas == nil {
		return nil, ErrNoFeeCap
	}
	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, tx.payload()); err != nil {
		return nil, err
	}
	return Keccak256([]byte{TxTypeDynamicFee}, buf.Bytes()), nil
}

// Sign signs the transaction with privateKey, filling YParity, R and S, and
// returns the signature as 65 bytes of R || S || recovery id.
func (tx *EthDynamicFeeTxSigner) Sign(privateKey *ecdsa.PrivateKey) (sig []byte, err error) {
	digest, err := tx.SigningHash()
	if err != nil {
		return nil, err
	}
	recoveryID, r, s := SignEcdsaRfc6979(privateKey, digest, sha256.New)

	sig = make([]byte, 65)
	copy(sig, padBytes(r.Bytes(), 32))
	copy(sig[32:], padBytes(s.Bytes(), 32))
	sig[64] = recoveryID

	tx.YParity = big.NewInt(int64(recoveryID))
	tx.R, tx.S = r, s
	return sig, nil
}

// EncodeRPL returns the signed transaction in the form
// eth_sendRawTransaction takes: the type byte, then the RLP of the payload
// with the three signature fields appended. The name is kept for symmetry
// with EthTxSigner.EncodeRPL.
func (tx *EthDynamicFeeTxSigner) EncodeRPL() (data []byte, err error) {
	if tx.YParity == nil || tx.R == nil || tx.S == nil {
		return nil, ErrNotSigned
	}
	buf := new(bytes.Buffer)
	full := append(tx.payload(), tx.YParity, tx.R, tx.S)
	if err := rlp.Encode(buf, full); err != nil {
		return nil, err
	}
	return append([]byte{TxTypeDynamicFee}, buf.Bytes()...), nil
}

// Hash is the transaction hash a node reports: the keccak-256 of the whole
// typed encoding, type byte included -- not the pre-signature SigningHash.
// Hashing the RLP alone, or hashing before the signature fields are appended,
// gives a different answer than the one a receipt is ever mined under.
func (tx *EthDynamicFeeTxSigner) Hash() ([]byte, error) {
	encoded, err := tx.EncodeRPL()
	if err != nil {
		return nil, err
	}
	return Keccak256(encoded), nil
}
