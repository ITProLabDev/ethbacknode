package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"
)

// The reference encodings below come from Foundry's cast (an independent
// implementation), the same oracle EthTxSigner is checked against in
// production use elsewhere in this codebase: RFC 6979 makes the signing nonce
// a function of the key and the digest, so a correct implementation
// reproduces another one byte for byte. These pin the signature itself, not
// merely the shape of the envelope.
//
//	cast mktx --private-key <key> --nonce N \
//	          --priority-gas-price P --gas-price M --gas-limit G \
//	          --value V --chain C <to> [data]
//
// cast's --gas-price is maxFeePerGas on a type-2 transaction and
// --priority-gas-price is the tip.
//
// devKey0/devKey1 are Anvil's well-known first two dev-account private keys
// (public, printed on every anvil startup banner; not sensitive).
const (
	devKey0 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	devKey1 = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

func mustKeyDF(t *testing.T, hexKey string) *ecdsa.PrivateKey {
	t.Helper()
	priv, _ := ECDSAKeysFromPrivateKeyBytes(mustHexDF(t, hexKey))
	if priv == nil {
		t.Fatalf("bad key %q", hexKey)
	}
	return priv
}

func mustHexDF(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		t.Fatalf("bad hex literal %q: %v", s, err)
	}
	return b
}

func bigFromDecDF(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad decimal %q", s)
	}
	return v
}

type dynamicFeeVector struct {
	name        string
	key         string
	nonce       uint64
	priorityFee *big.Int
	feeCap      *big.Int
	gas         uint64
	to          string // hex, empty for contract creation
	value       *big.Int
	data        string // hex
	chainID     int64
	raw         string // the whole signed transaction, from cast
}

func dynamicFeeVectors(t *testing.T) []dynamicFeeVector {
	t.Helper()
	return []dynamicFeeVector{
		{
			name:        "plain transfer",
			key:         devKey0,
			nonce:       9,
			priorityFee: big.NewInt(1_000_000_000),
			feeCap:      big.NewInt(20_000_000_000),
			gas:         21000,
			to:          "3535353535353535353535353535353535353535",
			value:       bigFromDecDF(t, "1000000000000000000"),
			chainID:     1,
			raw: "02f8730109843b9aca008504a817c800825208943535353535353535353535353535353535353535" +
				"880de0b6b3a764000080c001a010e3aef7fe1dd0f1d82bbc33c6b431d975e0f4471160babf09a2a84cc90e7eb9" +
				"a02d3b6bf9bd554d38298e970a7293232f9e61fb08cbba79ad04e370189a5b1763",
		},
		{
			name:        "zeros throughout",
			key:         devKey0,
			nonce:       0,
			priorityFee: new(big.Int),
			feeCap:      big.NewInt(1_000_000_000),
			gas:         21000,
			to:          "70997970c51812dc3a010c7d01b50e0d17dc79c8",
			value:       new(big.Int),
			chainID:     31337,
			raw: "02f868827a698080843b9aca008252089470997970c51812dc3a010c7d01b50e0d17dc79c88080c080" +
				"a02e7f97aa0dc2f2cf9c75963ed2acb3f765713310cae02d1baef82ebc31750e79" +
				"a064eb3151a1a9cbc1689b702a81b70c5944c7bf0170491fefdbda824468fd4f74",
		},
		{
			name:        "erc-20 transfer call",
			key:         devKey1,
			nonce:       3,
			priorityFee: big.NewInt(2_000_000_000),
			feeCap:      big.NewInt(30_000_000_000),
			gas:         100000,
			to:          "dac17f958d2ee523a2206206994597c13d831ec7",
			value:       new(big.Int),
			data: "a9059cbb" +
				"0000000000000000000000003535353535353535353535353535353535353535" +
				"000000000000000000000000000000000000000000000000000000000000007b",
			chainID: 1,
			raw: "02f8b1010384773594008506fc23ac00830186a094dac17f958d2ee523a2206206994597c13d831ec780" +
				"b844a9059cbb0000000000000000000000003535353535353535353535353535353535353535" +
				"000000000000000000000000000000000000000000000000000000000000007bc001" +
				"a09447355cd9ff0d657650488a911ddf786d5f6dc150742ca35fc0836d57295a26" +
				"a048bcb8a45c63ecb032dc451cb9e059f9deeea7a627c2b3fda1ada8d006dad2b2",
		},
		{
			name:        "contract creation",
			key:         devKey0,
			nonce:       1,
			priorityFee: big.NewInt(1_000_000_000),
			feeCap:      big.NewInt(20_000_000_000),
			gas:         500000,
			to:          "",
			value:       new(big.Int),
			data:        "6080604052",
			chainID:     1,
			raw: "02f85d0101843b9aca008504a817c8008307a1208080856080604052c080" +
				"a07d98c61dd159d4c0e3b821db3c7e0a0808b9cfed56473ab29024f12738e4fe77" +
				"a005d289f1b42435856582685c99192eb5e860f33de15a691d09a0db6cf9dd5306",
		},
		{
			name:        "two-byte chain id",
			key:         devKey0,
			nonce:       7,
			priorityFee: big.NewInt(100_000_000),
			feeCap:      big.NewInt(500_000_000),
			gas:         21000,
			to:          "3535353535353535353535353535353535353535",
			value:       big.NewInt(12345),
			chainID:     42161,
			raw: "02f86e82a4b1078405f5e100841dcd650082520894353535353535353535353535353535353535353582303980c080" +
				"a01fe4d6bf31cd5531952ec3a1f95fed018b19f877fa4a3834ea681ad458b41be4" +
				"a00dd2ae26a68067e37eb59dcd6a7bd448a96e506a68d7d8a3a7ba454bc4fa4aae",
		},
	}
}

func signerFor(t *testing.T, v dynamicFeeVector) *EthDynamicFeeTxSigner {
	t.Helper()

	var to []byte
	if v.to != "" {
		to = mustHexDF(t, v.to)
	}
	var data []byte
	if v.data != "" {
		data = mustHexDF(t, v.data)
	}

	tx := NewEthDynamicFeeTxSigner(v.nonce, v.priorityFee, v.feeCap, v.gas, to, v.value, data)
	if v.to == "" {
		// A creation names no recipient. NewEthDynamicFeeTxSigner takes the
		// address of its argument, so a nil slice arrives as a pointer to
		// nil, which RLP encodes as the empty string -- the same thing. Set
		// it explicitly so the intent is on the page.
		tx.To = nil
	}
	tx.SetChainId(big.NewInt(v.chainID))
	return tx
}

// TestDynamicFeeSigningMatchesCast is the whole point: our bytes are cast's bytes.
func TestDynamicFeeSigningMatchesCast(t *testing.T) {
	for _, v := range dynamicFeeVectors(t) {
		t.Run(v.name, func(t *testing.T) {
			tx := signerFor(t, v)
			if _, err := tx.Sign(mustKeyDF(t, v.key)); err != nil {
				t.Fatalf("Sign: %v", err)
			}
			encoded, err := tx.EncodeRPL()
			if err != nil {
				t.Fatalf("EncodeRPL: %v", err)
			}
			want, err := hex.DecodeString(v.raw)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, want) {
				t.Fatalf("signed transaction differs\n got %x\nwant %x", encoded, want)
			}
			if encoded[0] != TxTypeDynamicFee {
				t.Errorf("first byte is %#x, want the type byte %#x", encoded[0], TxTypeDynamicFee)
			}
		})
	}
}

// The hash a node reports is the keccak of the whole typed encoding, type
// byte included. Hashing the RLP alone gives a different answer and a caller
// that then waits for a receipt against it waits forever.
func TestDynamicFeeHashCoversTheTypeByte(t *testing.T) {
	v := dynamicFeeVectors(t)[0]
	tx := signerFor(t, v)
	if _, err := tx.Sign(mustKeyDF(t, v.key)); err != nil {
		t.Fatal(err)
	}

	got, err := tx.Hash()
	if err != nil {
		t.Fatal(err)
	}
	// From `cast decode-transaction` on the vector above, which reports the
	// hash alongside the fields it decoded.
	const want = "2e2910606d62f2fe7b730fb475c724fd2a1d8888207545a4e39ff54a41364da1"
	if hex.EncodeToString(got) != want {
		t.Fatalf("hash = %x, want %s", got, want)
	}

	// And it is not the hash of the RLP without the type byte, which is the
	// mistake this guards.
	encoded, err := tx.EncodeRPL()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, Keccak256(encoded[1:])) {
		t.Fatal("the hash was taken over the RLP without the type byte")
	}
}

// The parity bit stays a bit. Folding the chain id into it the way EIP-155
// does would make the transaction undecodable -- and on a chain with a large
// id it would produce a number no node would read as a parity.
func TestDynamicFeeParityIsNotAnEIP155V(t *testing.T) {
	v := dynamicFeeVectors(t)[4] // chain 42161
	tx := signerFor(t, v)
	if _, err := tx.Sign(mustKeyDF(t, v.key)); err != nil {
		t.Fatal(err)
	}
	if tx.YParity.Cmp(big.NewInt(1)) > 0 || tx.YParity.Sign() < 0 {
		t.Fatalf("yParity = %s, want 0 or 1", tx.YParity)
	}
}

// An access list is encoded as a list of [address, [storage keys]].
//
// cast mktx cannot build one, so the vector is verified the other way round:
// this encoding was fed to `cast decode-transaction`, which reported the
// address and both storage keys back, recovered 0xf39F...2266 as the signer,
// and gave the hash below.
func TestDynamicFeeAccessListEncoding(t *testing.T) {
	tx := NewEthDynamicFeeTxSigner(
		5,
		big.NewInt(1_000_000_000),
		big.NewInt(20_000_000_000),
		60000,
		mustHexDF(t, "3535353535353535353535353535353535353535"),
		big.NewInt(1),
		nil,
	)
	tx.SetChainId(big.NewInt(1))
	tx.AccessList = []AccessTuple{{
		Address: mustHexDF(t, "dac17f958d2ee523a2206206994597c13d831ec7"),
		StorageKeys: [][]byte{
			mustHexDF(t, "0000000000000000000000000000000000000000000000000000000000000000"),
			mustHexDF(t, "0000000000000000000000000000000000000000000000000000000000000001"),
		},
	}}

	if _, err := tx.Sign(mustKeyDF(t, devKey0)); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	encoded, err := tx.EncodeRPL()
	if err != nil {
		t.Fatalf("EncodeRPL: %v", err)
	}

	const want = "02f8c70105843b9aca008504a817c80082ea6094353535353535353535353535353535353535353501" +
		"80f85bf85994dac17f958d2ee523a2206206994597c13d831ec7f842" +
		"a00000000000000000000000000000000000000000000000000000000000000000" +
		"a00000000000000000000000000000000000000000000000000000000000000001" +
		"01a0752942ffcfe2e75b6299c102c30d4534b2ec0b9465cf87b319a4e3e191be0f4b" +
		"a01c96f45e03e930ced444e00085f37f7ededf8b208e5ca4c12a0733ae1dbb2824"
	if got := hex.EncodeToString(encoded); got != want {
		t.Fatalf("signed transaction differs\n got %s\nwant %s", got, want)
	}

	// The nesting, stated on its own so a failure says which part moved: one
	// list holding one pair, whose second item is a list of two 32-byte
	// strings.
	if !bytes.Contains(encoded, mustHexDF(t, "f85bf85994dac17f958d2ee523a2206206994597c13d831ec7f842")) {
		t.Fatalf("the access list is not encoded as [[address, [keys]]]: %x", encoded)
	}

	got, err := tx.Hash()
	if err != nil {
		t.Fatal(err)
	}
	const wantHash = "2b6415cc64f2ce11c3b9fc21d4d23cd66c53e9692ff1911810408473fd62e978"
	if hex.EncodeToString(got) != wantHash {
		t.Fatalf("hash = %x, want %s", got, wantHash)
	}
}

// Signing refuses rather than guesses when a figure that decides what the
// transaction can cost was left out.
func TestDynamicFeeSigningRefusesMissingFields(t *testing.T) {
	t.Run("no chain id", func(t *testing.T) {
		tx := NewEthDynamicFeeTxSigner(0, big.NewInt(1), big.NewInt(2), 21000, nil, nil, nil)
		if _, err := tx.SigningHash(); !errors.Is(err, ErrNoChainID) {
			t.Fatalf("err = %v, want ErrNoChainID", err)
		}
	})
	t.Run("no fee cap", func(t *testing.T) {
		tx := NewEthDynamicFeeTxSigner(0, big.NewInt(1), nil, 21000, nil, nil, nil)
		tx.SetChainId(big.NewInt(1))
		if _, err := tx.SigningHash(); !errors.Is(err, ErrNoFeeCap) {
			t.Fatalf("err = %v, want ErrNoFeeCap", err)
		}
	})
	t.Run("not signed", func(t *testing.T) {
		tx := NewEthDynamicFeeTxSigner(0, big.NewInt(1), big.NewInt(2), 21000, nil, nil, nil)
		tx.SetChainId(big.NewInt(1))
		if _, err := tx.EncodeRPL(); !errors.Is(err, ErrNotSigned) {
			t.Fatalf("err = %v, want ErrNotSigned", err)
		}
	})
}
