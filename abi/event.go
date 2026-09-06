package abi

import (
	"strings"
)

// Topic0 returns the 32-byte event signature hash
// keccak256("EventName(type1,type2,...)") used as topics[0] for a
// non-anonymous event. Unlike a method selector (first 4 bytes), an event
// topic uses the full 32-byte hash. Cached alongside the method selector by
// updateSignature, so decoding a log does not re-hash the entry's canonical
// signature on every call.
func (e *SmartContractAbiEntry) Topic0() [32]byte {
	e.updateSignature()
	return e.topic0
}

// DecodeLog decodes an event log (topics + data) into a DecodedEvent using the
// typed engine. topics[0] is the event signature; topics[1:] carry the indexed
// parameters in order; data carries the ABI-encoded non-indexed parameters.
//
// An indexed value type (address, uint*, int*, bool, bytesN) is decoded from
// its 32-byte topic slot. An indexed reference type (string, bytes, array —
// fixed or dynamic — or tuple) is NOT recoverable from the log: only
// keccak256(value) is stored in the topic, so it is returned as a 32-byte hash
// placeholder with its Type suffixed " (indexed)". That suffixed Type is for
// display only and is NOT a valid ABI type string — do not feed it back into
// parseType.
//
// Anonymous events are not supported (the ABI model carries no anonymous flag);
// a log for an anonymous event has no signature topic and will not match any
// registered event, so manager-level lookup resolves it to ErrUnknownEvent.
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

// GetEventByTopic0 finds the event entry whose signature hash equals topic0.
func (a *SmartContractAbi) GetEventByTopic0(topic0 [32]byte) (*SmartContractAbiEntry, error) {
	a._prepare()
	for _, entry := range a.Entries {
		if !strings.EqualFold(entry.Type, "Event") {
			continue
		}
		if entry.Topic0() == topic0 {
			return entry, nil
		}
	}
	return nil, ErrUnknownEvent
}

// DecodeLog finds the contract registered at contractAddress, matches the event
// by topics[0], and decodes the log. The returned DecodedEvent has Contract set
// to contractAddress as-is (case-preserving); the internal registry lookup is
// case-insensitive.
func (m *SmartContractsManager) DecodeLog(contractAddress string, topics [][32]byte, data []byte) (*DecodedEvent, error) {
	if len(topics) == 0 {
		return nil, ErrTopicCountMismatch
	}
	contract, err := m.GetSmartContractByAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	if contract.Abi == nil {
		return nil, ErrUnknownEvent
	}
	entry, err := contract.Abi.GetEventByTopic0(topics[0])
	if err != nil {
		return nil, err
	}
	ev, err := entry.DecodeLog(topics, data)
	if err != nil {
		return nil, err
	}
	ev.Contract = contractAddress
	return ev, nil
}
