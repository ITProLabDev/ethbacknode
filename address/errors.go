package address

import "errors"

// Error definitions for address operations.
var (
	// ErrAddressUnknown is returned when an address is not found in the pool.
	ErrAddressUnknown = errors.New("unknown address")
	// ErrAddressExists is returned when trying to add an existing address.
	ErrAddressExists = errors.New("address already exists")
	// ErrConfigStorageEmpty is returned when config storage is not set.
	ErrConfigStorageEmpty = errors.New("config storage not set")
	// ErrNoFreeAddresses is returned when no free addresses are available.
	ErrNoFreeAddresses = errors.New("no free addresses")
	// ErrInvalidAddressBytes is returned for invalid address byte length.
	ErrInvalidAddressBytes = errors.New("invalid address bytes")
	// ErrInvalidAddress is returned for malformed address strings.
	ErrInvalidAddress = errors.New("invalid address string")
	// ErrAddressBytesEmpty is returned when address bytes are empty.
	ErrAddressBytesEmpty = errors.New("address bytes empty")
	// ErrAddressStringEmpty is returned when address string is empty.
	ErrAddressStringEmpty = errors.New("address string empty")
	// ErrPrivateKeyEmpty is returned when private key is required but empty.
	ErrPrivateKeyEmpty = errors.New("private key empty")
	// ErrAddressPrivateKeyMismatch is returned when address doesn't match private key.
	ErrAddressPrivateKeyMismatch = errors.New("address and private key mismatch")
	// ErrInvalidMnemonicLen is returned for invalid BIP-39 mnemonic length.
	ErrInvalidMnemonicLen = errors.New("invalid mnemonic length")
)
