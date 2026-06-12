package abi

import (
	"strings"

	"github.com/ITProLabDev/ethbacknode/crypto"
)

// Topic0 returns the 32-byte event signature hash
// keccak256("EventName(type1,type2,...)") used as topics[0] for a
// non-anonymous event. Unlike a method selector (first 4 bytes), an event
// topic uses the full 32-byte hash.
func (e *SmartContractAbiEntry) Topic0() [32]byte {
	h := crypto.Keccak256([]byte(e.canonicalSignature()))
	var out [32]byte
	copy(out[:], h)
	return out
}

// DecodeLog decodes an event log (topics + data) into a DecodedEvent using the
// typed engine. topics[0] is the event signature; topics[1:] carry the indexed
// parameters in order; data carries the ABI-encoded non-indexed parameters.
//
// A static indexed parameter is decoded from its 32-byte topic slot. A dynamic
// indexed parameter (string, bytes, array, tuple) is NOT recoverable from the
// log — only keccak256(value) is stored in the topic — so it is returned as a
// 32-byte hash placeholder with its Type suffixed " (indexed)".
func (e *SmartContractAbiEntry) DecodeLog(topics [][32]byte, data []byte) (*DecodedEvent, error) {
	if !strings.EqualFold(e.Type, "Event") {
		return nil, ErrNotAnEvent
	}

	// Count indexed params and verify the topic count.
	indexedCount := 0
	for _, in := range e.Inputs {
		if in.Indexed {
			indexedCount++
		}
	}
	if len(topics) != indexedCount+1 { // +1 for topics[0] (the signature)
		return nil, ErrTopicCountMismatch
	}

	// Split inputs into indexed (decoded from topics) and non-indexed (decoded
	// from data, preserving order).
	nonIndexedTypes := make([]abiType, 0, len(e.Inputs))
	for _, in := range e.Inputs {
		if in.Indexed {
			continue
		}
		typ, err := parseType(in.Type, in.Components)
		if err != nil {
			return nil, err
		}
		nonIndexedTypes = append(nonIndexedTypes, typ)
	}

	dataVals, err := decodeParams(nonIndexedTypes, data)
	if err != nil {
		return nil, err
	}

	out := &DecodedEvent{Name: e.Name, Inputs: make([]DecodedValue, len(e.Inputs))}
	topicPos := 1 // topics[0] is the signature
	dataPos := 0
	for i, in := range e.Inputs {
		if in.Indexed {
			dv, derr := decodeIndexedTopic(in, topics[topicPos])
			if derr != nil {
				return nil, derr
			}
			dv.Name = in.Name
			out.Inputs[i] = dv
			topicPos++
		} else {
			dv := dataVals[dataPos]
			dv.Name = in.Name
			out.Inputs[i] = dv
			dataPos++
		}
	}
	return out, nil
}

// decodeIndexedTopic decodes a single indexed parameter from its 32-byte topic.
// Static types are decoded normally; dynamic types are not recoverable and are
// returned as a 32-byte hash placeholder.
func decodeIndexedTopic(in *SmartContractAbiEntryInput, topic [32]byte) (DecodedValue, error) {
	typ, err := parseType(in.Type, in.Components)
	if err != nil {
		return DecodedValue{}, err
	}
	// Per the Solidity ABI spec, indexed reference types (arrays — fixed or
	// dynamic — tuples/structs, string, bytes) are stored in the topic as
	// keccak256(value); only value types are stored directly.
	switch typ.kind {
	case kindArray, kindSlice, kindTuple, kindString, kindBytes:
		hash := make([]byte, 32)
		copy(hash, topic[:])
		return DecodedValue{Type: typ.canonical() + " (indexed)", Value: hash}, nil
	}
	return decodeValue(typ, topic[:], 0)
}
