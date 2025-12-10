package address

import "errors"

var (
	ErrAddressUnknown            = errors.New("unknown address")
	ErrAddressExists             = errors.New("address already exists")
	ErrConfigStorageEmpty        = errors.New("config storage not set")
	ErrNoFreeAddresses           = errors.New("no free addresses")
	ErrInvalidAddressBytes       = errors.New("invalid address bytes")
	ErrInvalidAddress            = errors.New("invalid address string")
	ErrAddressBytesEmpty         = errors.New("address bytes empty")
	ErrAddressStringEmpty        = errors.New("address string empty")
	ErrPrivateKeyEmpty           = errors.New("private key empty")
	ErrAddressPrivateKeyMismatch = errors.New("address and private key mismatch")
	ErrInvalidMnemonicLen        = errors.New("invalid mnemonic length")
)
