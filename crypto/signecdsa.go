package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"hash"
	"math/big"
)

// SignEcdsaRfc6979 signs an arbitrary length hash (which should be the result of
// hashing a larger message) using the private key, priv. It returns the
// signature as a pair of integers.
//
// Note that FIPS 186-3 section 4.6 specifies that the hash should be truncated
// to the byte-length of the subgroup. This function does not perform that
// truncation itself.

func SignEcdsaRfc6979(priv *ecdsa.PrivateKey, hash []byte, alg func() hash.Hash) (v byte, r, s *big.Int) {
	c := priv.Curve
	N := c.Params().N
	v = 0
	generateSecret(N, priv.D, alg, hash, func(k *big.Int) bool {
		var y *big.Int
		inv := new(big.Int).ModInverse(k, N)
		r, y = priv.Curve.ScalarBaseMult(k.Bytes())
		// 1) The X coord of the random point is < N and its Y coord even
		// 2) The X coord of the random point is < N and its Y coord is odd
		// 3) The X coord of the random point is >= N and its Y coord is even
		// 4) The X coord of the random point is >= N and its Y coord is odd
		v = byte(y.Bit(0))
		if r.Sign() == 0 {
			return false
		}
		e := hashToInt(hash, c)
		s = new(big.Int).Mul(priv.D, r)
		s.Add(s, e)
		s.Mul(s, inv)
		sR := new(big.Int).Mod(s, N)
		if new(big.Int).Div(N, big.NewInt(2)).Cmp(sR) == -1 {
			s.Neg(s)
			s.Mod(s, N)
			v ^= 0x01
		} else {
			s = sR
		}
		return s.Sign() != 0
	})
	return v, r, s
}

func SignEcdsaRfc6979Bytes(privateKey *ecdsa.PrivateKey, hash []byte, alg func() hash.Hash) (sig []byte) {
	v, r, s := SignEcdsaRfc6979(privateKey, hash, alg)
	sig = make([]byte, 65)
	copy(sig[0:], padBytes(r.Bytes(), 32))
	copy(sig[32:], padBytes(s.Bytes(), 32))
	sig[64] = v
	return
}

func padBytes(slice []byte, length int) []byte {
	if len(slice) < length {
		slice = append(make([]byte, length-len(slice)), slice...)
	}
	return slice
}

// copied from crypto/ecdsa
func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}
