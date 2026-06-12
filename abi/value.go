package abi

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
)

// DecodedValue is one decoded ABI value. Value holds one of:
// *big.Int (uint*/int*), bool, []byte (address/bytesN/bytes),
// string, or []DecodedValue (arrays, slices, tuples).
type DecodedValue struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// MarshalJSON renders a DecodedValue in a client-friendly, JSON-safe shape that
// is stable regardless of the Go type held in Value:
//   - []byte (address, bytesN, bytes, indexed-reference hash) -> "0x"+lowercase
//     hex string (an empty slice becomes "0x"); never base64.
//   - *big.Int (uint*/int*) -> a DECIMAL STRING, not a JSON number, so values
//     above 2^53 survive JavaScript's JSON.parse without precision loss.
//   - bool, string -> native JSON.
//   - []DecodedValue (arrays, slices, tuples) -> a JSON array, each element
//     marshaled by this same method (recursion).
//
// The in-memory Value type is unchanged; only the JSON wire form is transformed.
func (d DecodedValue) MarshalJSON() ([]byte, error) {
	type wire struct {
		Name  string `json:"name,omitempty"`
		Type  string `json:"type"`
		Value any    `json:"value"`
	}
	return json.Marshal(wire{Name: d.Name, Type: d.Type, Value: jsonSafeValue(d.Value)})
}

// jsonSafeValue converts a decoded Value into a JSON-safe representation:
// byte slices to 0x-hex strings, big integers to decimal strings, and nested
// decoded values recursively. Other types pass through unchanged.
func jsonSafeValue(v any) any {
	switch val := v.(type) {
	case []byte:
		return "0x" + hex.EncodeToString(val)
	case *big.Int:
		if val == nil {
			return nil
		}
		return val.String()
	case []DecodedValue:
		// Marshaled element-wise via DecodedValue.MarshalJSON; return as-is.
		return val
	default:
		return v
	}
}

// DecodedCall is a decoded method invocation: the method name and its inputs.
type DecodedCall struct {
	Method string         `json:"method"`
	Inputs []DecodedValue `json:"inputs"`
}

// DecodedEvent is a decoded log event: its name, the emitting contract, and
// the decoded indexed + non-indexed parameters in ABI order.
type DecodedEvent struct {
	Name     string         `json:"name"`
	Contract string         `json:"contract,omitempty"`
	Inputs   []DecodedValue `json:"inputs"`
}
