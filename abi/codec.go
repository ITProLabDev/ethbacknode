package abi

import (
	"fmt"
	"math/big"
)

var bigOne = big.NewInt(1)

// leftPad32 right-aligns b into a 32-byte slot.
func leftPad32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) >= 32 {
		copy(out, b[len(b)-32:])
		return out
	}
	copy(out[32-len(b):], b)
	return out
}

// rightPad pads b on the right to a multiple of 32 bytes.
func rightPad(b []byte) []byte {
	if len(b)%32 == 0 {
		return b
	}
	out := make([]byte, ((len(b)/32)+1)*32)
	copy(out, b)
	return out
}

// encodeParams encodes a sequence of values using head/tail offset encoding.
func encodeParams(types []abiType, values []any) ([]byte, error) {
	if len(types) != len(values) {
		return nil, ErrSmartContractMethodParamsCountMismatch
	}
	heads := make([][]byte, len(types))
	tails := make([][]byte, len(types))
	headLen := 0
	for i := range types {
		enc, err := encodeValue(types[i], values[i])
		if err != nil {
			return nil, err
		}
		if types[i].isDynamic() {
			tails[i] = enc
			headLen += 32
		} else {
			heads[i] = enc
			headLen += len(enc)
		}
	}
	tailSeen := 0
	for i := range types {
		if types[i].isDynamic() {
			heads[i] = leftPad32(big.NewInt(int64(headLen + tailSeen)).Bytes())
			tailSeen += len(tails[i])
		}
	}
	out := make([]byte, 0, headLen+tailSeen)
	for i := range heads {
		out = append(out, heads[i]...)
	}
	for i := range tails {
		out = append(out, tails[i]...)
	}
	return out, nil
}

// encodeValue encodes a single value according to its type.
func encodeValue(t abiType, v any) ([]byte, error) {
	switch t.kind {
	case kindUint:
		n, err := asBigInt(v)
		if err != nil {
			return nil, err
		}
		if n.Sign() < 0 {
			return nil, fmt.Errorf("%w: uint cannot be negative", ErrInvalidParamsData)
		}
		return leftPad32(n.Bytes()), nil
	case kindInt:
		n, err := asBigInt(v)
		if err != nil {
			return nil, err
		}
		return encodeInt(n), nil
	case kindBool:
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("%w: bool expected", ErrInvalidParamsData)
		}
		out := make([]byte, 32)
		if b {
			out[31] = 1
		}
		return out, nil
	case kindAddress:
		b, ok := v.([]byte)
		if !ok || len(b) != 20 {
			return nil, fmt.Errorf("%w: address must be 20 bytes", ErrInvalidParamsData)
		}
		return leftPad32(b), nil
	case kindFixedBytes:
		b, ok := v.([]byte)
		if !ok || len(b) != t.size {
			return nil, fmt.Errorf("%w: bytes%d expected", ErrInvalidParamsData, t.size)
		}
		out := make([]byte, 32)
		copy(out, b) // left-aligned
		return out, nil
	}
	return nil, fmt.Errorf("%w: unsupported type %q", ErrInvalidParamsData, t.canonical())
}

// encodeInt encodes a (possibly negative) integer as a 32-byte two's complement.
func encodeInt(n *big.Int) []byte {
	if n.Sign() >= 0 {
		return leftPad32(n.Bytes())
	}
	mod := new(big.Int).Lsh(bigOne, 256)
	tc := new(big.Int).Add(mod, n) // n is negative
	return leftPad32(tc.Bytes())
}

func asBigInt(v any) (*big.Int, error) {
	switch x := v.(type) {
	case *big.Int:
		return x, nil
	case int:
		return big.NewInt(int64(x)), nil
	case int64:
		return big.NewInt(x), nil
	case uint64:
		return new(big.Int).SetUint64(x), nil
	default:
		return nil, fmt.Errorf("%w: integer expected, got %T", ErrInvalidParamsData, v)
	}
}

// decodeParams decodes a head/tail-encoded sequence. All offsets in block are
// relative to block[0].
func decodeParams(types []abiType, block []byte) ([]DecodedValue, error) {
	out := make([]DecodedValue, len(types))
	head := 0
	for i := range types {
		if types[i].isDynamic() {
			if head+32 > len(block) {
				return nil, ErrInvalidParamsData
			}
			offBig := new(big.Int).SetBytes(block[head : head+32])
			if !offBig.IsInt64() {
				return nil, ErrInvalidParamsData
			}
			off := int(offBig.Int64())
			if off < 0 || off > len(block) {
				return nil, ErrInvalidParamsData
			}
			val, err := decodeValue(types[i], block, off)
			if err != nil {
				return nil, err
			}
			out[i] = val
			head += 32
		} else {
			sz := types[i].staticSize()
			if head+sz > len(block) {
				return nil, ErrInvalidParamsData
			}
			val, err := decodeValue(types[i], block, head)
			if err != nil {
				return nil, err
			}
			out[i] = val
			head += sz
		}
	}
	return out, nil
}

// decodeValue decodes a single value of type t located at offset at within block.
func decodeValue(t abiType, block []byte, at int) (DecodedValue, error) {
	dv := DecodedValue{Type: t.canonical()}
	if at < 0 {
		return dv, ErrInvalidParamsData
	}
	switch t.kind {
	case kindUint:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = new(big.Int).SetBytes(block[at : at+32])
		return dv, nil
	case kindInt:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = decodeInt(block[at : at+32])
		return dv, nil
	case kindBool:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		dv.Value = block[at+31] != 0
		return dv, nil
	case kindAddress:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		b := make([]byte, 20)
		copy(b, block[at+12:at+32])
		dv.Value = b
		return dv, nil
	case kindFixedBytes:
		if at+32 > len(block) {
			return dv, ErrInvalidParamsData
		}
		b := make([]byte, t.size)
		copy(b, block[at:at+t.size])
		dv.Value = b
		return dv, nil
	}
	return dv, fmt.Errorf("%w: unsupported type %q", ErrInvalidParamsData, t.canonical())
}

// decodeInt reads a 32-byte two's complement integer.
func decodeInt(slot []byte) *big.Int {
	v := new(big.Int).SetBytes(slot)
	if v.Bit(255) == 1 {
		mod := new(big.Int).Lsh(bigOne, 256)
		v.Sub(v, mod)
	}
	return v
}
