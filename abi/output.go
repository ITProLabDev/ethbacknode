package abi

// DecodeOutputs decodes ABI-encoded return data against this entry's declared
// outputs, returning the values in declaration order with their names. Unlike
// DecodeInputsTyped there is no 4-byte selector to skip — eth_call returns the
// bare encoded output block. Tuple outputs are not supported (the output model
// carries no components); such an ABI is rejected at import time (M3).
func (e *SmartContractAbiEntry) DecodeOutputs(data []byte) ([]DecodedValue, error) {
	if len(e.Outputs) == 0 {
		return nil, nil
	}
	types := make([]abiType, len(e.Outputs))
	for i, out := range e.Outputs {
		typ, err := parseType(out.Type, nil)
		if err != nil {
			return nil, err
		}
		types[i] = typ
	}
	vals, err := decodeParams(types, data)
	if err != nil {
		return nil, err
	}
	for i := range vals {
		if i < len(e.Outputs) {
			vals[i].Name = e.Outputs[i].Name
		}
	}
	return vals, nil
}
