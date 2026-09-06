package ethclient

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ITProLabDev/ethbacknode/crypto"
)

// devKey0 is Anvil's well-known first dev-account private key (public,
// printed on every anvil startup banner; not sensitive). Same key and vector
// crypto's own dynamic-fee signer tests are checked against.
const devKey0 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func mustDevKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	b, err := hex.DecodeString(devKey0)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := crypto.ECDSAKeysFromPrivateKeyBytes(b)
	if pk == nil {
		t.Fatal("no key derived")
	}
	return pk
}

func bigFromDec(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad decimal %q", s)
	}
	return v
}

// This is the exact "plain transfer" vector crypto/eth_dynamic_fee_tx_signer_test.go
// checks against cast: nonce 9, tip 1 gwei, cap 20 gwei, chain 1, 1 ETH to
// 0x3535...35. Reusing it here proves signTransfer wires resolveFees' output
// into the signer correctly, not just that the signer itself is correct.
func TestSignTransferBuildsTheDynamicFeeEnvelopeCastVerifies(t *testing.T) {
	to, err := hex.DecodeString("3535353535353535353535353535353535353535")
	if err != nil {
		t.Fatal(err)
	}
	fees := txFees{
		dynamic:              true,
		maxPriorityFeePerGas: big.NewInt(1_000_000_000),
		maxFeePerGas:         big.NewInt(20_000_000_000),
	}

	got, err := signTransfer(mustDevKey(t), big.NewInt(1), 9, to, bigFromDec(t, "1000000000000000000"), fees, 21000)
	if err != nil {
		t.Fatalf("signTransfer: %v", err)
	}

	const want = "02f8730109843b9aca008504a817c800825208943535353535353535353535353535353535353535" +
		"880de0b6b3a764000080c001a010e3aef7fe1dd0f1d82bbc33c6b431d975e0f4471160babf09a2a84cc90e7eb9" +
		"a02d3b6bf9bd554d38298e970a7293232f9e61fb08cbba79ad04e370189a5b1763"
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, wantBytes) {
		t.Fatalf("signed transaction differs\n got %x\nwant %x", got, wantBytes)
	}
}

// A legacy fees value must produce the pre-existing nine-field envelope, not
// a typed one -- checked by comparing against crypto.EthTxSigner built the
// same way, and by the first byte being an RLP list header (0xc0 or above)
// rather than the type-2 byte.
func TestSignTransferBuildsALegacyEnvelopeWhenFeesAreLegacy(t *testing.T) {
	to, err := hex.DecodeString("3535353535353535353535353535353535353535")
	if err != nil {
		t.Fatal(err)
	}
	fees := txFees{gasPrice: big.NewInt(20_000_000_000)}
	amount := big.NewInt(1_000_000_000_000_000_000)

	got, err := signTransfer(mustDevKey(t), big.NewInt(1), 9, to, amount, fees, 21000)
	if err != nil {
		t.Fatalf("signTransfer: %v", err)
	}
	if got[0] < 0xc0 {
		t.Fatalf("first byte is %#x, want an RLP list header (>= 0xc0), not a typed envelope", got[0])
	}

	want := &crypto.EthTxSigner{
		Nonce:    9,
		GasPrice: big.NewInt(20_000_000_000),
		Gas:      21000,
		To:       &to,
		Value:    amount,
	}
	want.SetChainId(big.NewInt(1))
	want.Sign(mustDevKey(t))
	wantBytes, err := want.EncodeRPL()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, wantBytes) {
		t.Fatalf("signed transaction differs from an equivalently-built EthTxSigner\n got %x\nwant %x", got, wantBytes)
	}
}
