package abi

// DecodedValue is one decoded ABI value. Value holds one of:
// *big.Int (uint*/int*), bool, []byte (address/bytesN/bytes),
// string, or []DecodedValue (arrays, slices, tuples).
type DecodedValue struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Value any    `json:"value"`
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
