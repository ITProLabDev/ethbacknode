// Package address provides address generation, storage, and subscription management
// for blockchain addresses. It supports BIP-39/44 HD wallet derivation.
package address

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"github.com/ITProLabDev/ethbacknode/crypto"
	"github.com/ITProLabDev/ethbacknode/crypto/secp256k1"
)

// Address represents a blockchain address with its associated data.
// Supports both regular addresses and HD wallet addresses (BIP-39/44).
type Address struct {
	// Address is the human-readable address string (e.g., 0x...).
	Address string `json:"address" storm:"id"`
	// AddressBytes is the raw 20-byte address.
	AddressBytes []byte `json:"addressBytes" storm:"index"`
	// PrivateKey is the 32-byte private key (nil for watch-only).
	PrivateKey []byte `json:"privateKey"`
	// Master indicates if this is a master/root address.
	Master bool `json:"master"`

	// Subscribed indicates if the address is subscribed for notifications.
	Subscribed bool `json:"subscribed"`
	// ServiceId is the subscriber's service identifier.
	ServiceId int `json:"serviceId"`
	// UserId is the subscriber's user identifier.
	UserId int64 `json:"userId"`
	// InvoiceId is the associated invoice identifier.
	InvoiceId int64 `json:"invoiceId"`
	// WatchOnly indicates the address has no private key.
	WatchOnly bool `json:"watchOnly"`

	// Bip39Support indicates BIP-39 mnemonic support.
	Bip39Support bool `json:"bip39Support,omitempty"`
	// Bip39Mnemonic stores the BIP-39 mnemonic words.
	Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
}

// AddressCodec defines the interface for address encoding/decoding.
// Implementations are chain-specific (e.g., Ethereum, Bitcoin).
type AddressCodec interface {
	// EncodeBytesToAddress converts raw bytes to address string.
	EncodeBytesToAddress(addressBytes []byte) (string, error)
	// DecodeAddressToBytes converts address string to raw bytes.
	DecodeAddressToBytes(address string) ([]byte, error)
	// PrivateKeyToAddress derives address from private key.
	PrivateKeyToAddress(privateKey []byte) (string, []byte, error)
	// IsValid checks if an address string is valid.
	IsValid(address string) bool
}

// String returns the address string representation.
func (a *Address) String() string {
	return a.Address
}

// GetKey returns the address bytes as storage key.
// Implements storage.Key interface.
func (a *Address) GetKey() []byte {
	return a.AddressBytes
}

// Encode serializes the address to bytes using gob encoding.
// Implements storage.Data interface.
func (m *Address) Encode() []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	enc.Encode(m)
	return b.Bytes()
}

// Decode deserializes the address from bytes.
// Implements storage.Data interface.
func (m *Address) Decode(b []byte) error {
	buf := bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)
	return enc.Decode(m)
}

// NewAddressRecordFill creates a new address record with custom initialization.
// The fill function allows setting additional fields before validation.
func (p *Manager) NewAddressRecordFill(addressString string, fill func(a *Address)) (addressRecord *Address, err error) {
	addressBytes, err := p.addressCodec.DecodeAddressToBytes(addressString)
	if err != nil {
		return nil, err
	}
	addressRecord = &Address{
		AddressBytes: addressBytes,
		Address:      addressString,
	}
	fill(addressRecord)
	if err := p.isRecordValid(addressRecord); err != nil {
		return nil, err
	}
	return addressRecord, nil
}

// NewAddressRecord creates a new address record with the given address and private key.
func (p *Manager) NewAddressRecord(addressString string, privateKey []byte) (addressRecord *Address, err error) {
	addressBytes, err := p.addressCodec.DecodeAddressToBytes(addressString)
	if err != nil {
		return nil, err
	}
	addressRecord = &Address{
		AddressBytes: addressBytes,
		Address:      addressString,
		PrivateKey:   privateKey,
	}
	if err := p.isRecordValid(addressRecord); err != nil {
		return nil, err
	}
	return addressRecord, nil
}

// isRecordValid validates that an address record has all required fields.
func (p *Manager) isRecordValid(address *Address) (err error) {
	if address.Address == "" {
		return ErrAddressStringEmpty
	}
	if len(address.AddressBytes) == 0 {
		return ErrAddressBytesEmpty
	}
	if !address.WatchOnly && len(address.PrivateKey) == 0 {
		return ErrPrivateKeyEmpty
	}
	return nil
}
// preLoadAddresses loads all addresses from storage into memory.
// Separates addresses into all addresses and free (unsubscribed) addresses.
func (p *Manager) preLoadAddresses() (err error) {
	err = p.db.ReadAll(func(raw []byte) (err error) {
		address := &Address{}
		err = address.Decode(raw)
		if err != nil {
			return err
		}
		p.allAddresses[address.Address] = address
		if !address.Subscribed {
			p.freeAddresses[address.Address] = address
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.updatePool()
	return nil
}

// createNewAddress generates a new random address with private key.
func (p *Manager) createNewAddress() (addressRecord *Address, err error) {
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, err
	}
	addressStr, addressBytes, err := p.addressCodec.PrivateKeyToAddress(privateKey)
	if err != nil {
		return nil, err
	}
	addressRecord = &Address{
		Address:      addressStr,
		AddressBytes: addressBytes,
		PrivateKey:   privateKey,
	}
	return addressRecord, nil
}

// generatePrivateKey generates a new random ECDSA private key using secp256k1 curve.
func generatePrivateKey() (privateKey []byte, err error) {
	ecdsaKey, err := ecdsa.GenerateKey(secp256k1.P256k1(), rand.Reader)
	if err != nil {
		return nil, err
	}
	privateKey = crypto.BytesFromECDSAPrivateKey(ecdsaKey)
	return privateKey, nil
}
