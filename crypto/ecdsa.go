package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/crypto/secp256k1"
	"math/big"
)

func PubKeyToAddressBytes(p ecdsa.PublicKey) []byte {
	pubBytes := ECDSAPublicKeyToBytes(&p)
	return Keccak256(pubBytes[1:])[12:]
}

func ECDSAPublicKeyToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(secp256k1.P256k1(), pub.X, pub.Y)
}

func ECDSAPublicKeyCompressedToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.MarshalCompressed(secp256k1.P256k1(), pub.X, pub.Y)
}

func generateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(secp256k1.P256k1(), rand.Reader)
}

func ECDSAKeysFromPrivateKeyHex(pk string) (priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey, err error) {
	pkBytes, err := hexnum.ParseHexBytes(pk)
	if err != nil {
		return nil, nil, err
	}
	priv, pub = ECDSAKeysFromPrivateKeyBytes(pkBytes)
	return priv, pub, nil
}

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

func BytesFromECDSAPrivateKey(privateKey *ecdsa.PrivateKey) []byte {
	if privateKey == nil {
		return nil
	}
	return paddedBigBytes(privateKey.D, privateKey.Params().BitSize/8)
}
