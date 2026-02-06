// Package crypto provides cryptographic operations for Ethereum transactions.
// It implements ECDSA signing, Keccak-256 hashing, and key management using
// the secp256k1 elliptic curve. This package supports RFC 6979 deterministic
// signatures and Ethereum transaction signing.
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/crypto/secp256k1"
	"math/big"
)

// PubKeyToAddressBytes derives an Ethereum address from an ECDSA public key.
// Returns the last 20 bytes of the Keccak-256 hash of the public key.
func PubKeyToAddressBytes(p ecdsa.PublicKey) []byte {
	pubBytes := ECDSAPublicKeyToBytes(&p)
	return Keccak256(pubBytes[1:])[12:]
}

// ECDSAPublicKeyToBytes serializes an ECDSA public key to uncompressed format.
// Returns 65 bytes: 0x04 prefix + 32-byte X + 32-byte Y coordinates.
func ECDSAPublicKeyToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(secp256k1.P256k1(), pub.X, pub.Y)
}

// ECDSAPublicKeyCompressedToBytes serializes an ECDSA public key to compressed format.
// Returns 33 bytes: 0x02 or 0x03 prefix + 32-byte X coordinate.
func ECDSAPublicKeyCompressedToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.MarshalCompressed(secp256k1.P256k1(), pub.X, pub.Y)
}

// generateKey generates a new random ECDSA private key on the secp256k1 curve.
func generateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(secp256k1.P256k1(), rand.Reader)
}

// ECDSAKeysFromPrivateKeyHex derives ECDSA key pair from a hex-encoded private key.
func ECDSAKeysFromPrivateKeyHex(pk string) (priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey, err error) {
	pkBytes, err := hexnum.ParseHexBytes(pk)
	if err != nil {
		return nil, nil, err
	}
	priv, pub = ECDSAKeysFromPrivateKeyBytes(pkBytes)
	return priv, pub, nil
}

// ECDSAKeysFromPrivateKeyBytes derives ECDSA key pair from raw private key bytes.
// Uses the secp256k1 curve for key derivation.
func ECDSAKeysFromPrivateKeyBytes(pk []byte) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	curve := secp256k1.P256k1()
	x, y := curve.ScalarBaseMult(pk)
	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}
	return priv, &priv.PublicKey
}

// BytesFromECDSAPrivateKey extracts the raw bytes from an ECDSA private key.
// Returns the private key D value padded to the curve's bit size.
func BytesFromECDSAPrivateKey(privateKey *ecdsa.PrivateKey) []byte {
	if privateKey == nil {
		return nil
	}
	return paddedBigBytes(privateKey.D, privateKey.Params().BitSize/8)
}
