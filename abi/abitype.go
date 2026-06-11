package abi

import (
	"fmt"
	"strconv"
	"strings"
)

// kind enumerates the ABI type families the typed codec understands.
type kind int

const (
	kindUint kind = iota
	kindInt
	kindBool
	kindAddress
	kindFixedBytes // bytesN, 1..32
	kindBytes      // dynamic bytes
	kindString
	kindArray // T[N]
	kindSlice // T[]
	kindTuple
)

// abiType is a parsed ABI type. For uint/int, size is the bit width; for
// fixedBytes, size is N; for array, size is the fixed length. elem is the
// element type for array/slice; fields/names describe tuple components.
type abiType struct {
	kind   kind
	size   int
	elem   *abiType
	fields []abiType
	names  []string
}

// parseType parses a Solidity ABI type string. For tuples (type "tuple",
// "tuple[]", "tuple[N]"), components supplies the component definitions.
func parseType(s string, components []*SmartContractAbiEntryInput) (abiType, error) {
	if s == "" {
		return abiType{}, fmt.Errorf("%w: empty type", ErrInvalidParamsData)
	}
	// Array/slice suffix at the outermost level.
	if strings.HasSuffix(s, "]") {
		open := strings.LastIndexByte(s, '[')
		if open < 0 {
			return abiType{}, fmt.Errorf("%w: malformed array %q", ErrInvalidParamsData, s)
		}
		inner := s[:open]
		between := s[open+1 : len(s)-1]
		elem, err := parseType(inner, components)
		if err != nil {
			return abiType{}, err
		}
		if between == "" {
			return abiType{kind: kindSlice, elem: &elem}, nil
		}
		n, err := strconv.Atoi(between)
		if err != nil || n <= 0 {
			return abiType{}, fmt.Errorf("%w: bad array size %q", ErrInvalidParamsData, between)
		}
		return abiType{kind: kindArray, size: n, elem: &elem}, nil
	}

	switch {
	case s == "bool":
		return abiType{kind: kindBool}, nil
	case s == "address":
		return abiType{kind: kindAddress}, nil
	case s == "string":
		return abiType{kind: kindString}, nil
	case s == "bytes":
		return abiType{kind: kindBytes}, nil
	case strings.HasPrefix(s, "bytes"):
		n, err := strconv.Atoi(s[len("bytes"):])
		if err != nil || n < 1 || n > 32 {
			return abiType{}, fmt.Errorf("%w: bad fixed bytes %q", ErrInvalidParamsData, s)
		}
		return abiType{kind: kindFixedBytes, size: n}, nil
	case strings.HasPrefix(s, "uint"):
		return parseIntType(s, "uint", kindUint)
	case strings.HasPrefix(s, "int"):
		return parseIntType(s, "int", kindInt)
	case s == "tuple":
		return parseTuple(components)
	}
	return abiType{}, fmt.Errorf("%w: unknown type %q", ErrInvalidParamsData, s)
}

func parseIntType(s, prefix string, k kind) (abiType, error) {
	rest := s[len(prefix):]
	if rest == "" {
		return abiType{kind: k, size: 256}, nil
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 8 || n > 256 || n%8 != 0 {
		return abiType{}, fmt.Errorf("%w: bad int width %q", ErrInvalidParamsData, s)
	}
	return abiType{kind: k, size: n}, nil
}

func parseTuple(components []*SmartContractAbiEntryInput) (abiType, error) {
	t := abiType{kind: kindTuple}
	for _, c := range components {
		ft, err := parseType(c.Type, c.Components)
		if err != nil {
			return abiType{}, err
		}
		t.fields = append(t.fields, ft)
		t.names = append(t.names, c.Name)
	}
	return t, nil
}

// isDynamic reports whether the type has no fixed encoded width.
func (t abiType) isDynamic() bool {
	switch t.kind {
	case kindString, kindBytes, kindSlice:
		return true
	case kindArray:
		return t.elem.isDynamic()
	case kindTuple:
		for _, f := range t.fields {
			if f.isDynamic() {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// staticSize returns the encoded byte width of a static type. Result is
// undefined (0) for dynamic types; callers must check isDynamic first.
func (t abiType) staticSize() int {
	switch t.kind {
	case kindArray:
		return t.size * t.elem.staticSize()
	case kindTuple:
		total := 0
		for _, f := range t.fields {
			total += f.staticSize()
		}
		return total
	default:
		return 32
	}
}

// canonical returns the canonical type string used in function/event
// signatures (e.g. "uint256", "(uint256,address)[]").
func (t abiType) canonical() string {
	switch t.kind {
	case kindUint:
		return "uint" + strconv.Itoa(t.size)
	case kindInt:
		return "int" + strconv.Itoa(t.size)
	case kindBool:
		return "bool"
	case kindAddress:
		return "address"
	case kindFixedBytes:
		return "bytes" + strconv.Itoa(t.size)
	case kindBytes:
		return "bytes"
	case kindString:
		return "string"
	case kindSlice:
		return t.elem.canonical() + "[]"
	case kindArray:
		return t.elem.canonical() + "[" + strconv.Itoa(t.size) + "]"
	case kindTuple:
		parts := make([]string, len(t.fields))
		for i, f := range t.fields {
			parts[i] = f.canonical()
		}
		return "(" + strings.Join(parts, ",") + ")"
	default:
		return ""
	}
}
