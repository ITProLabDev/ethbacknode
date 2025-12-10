package seedphrase

import "errors"

var (
	ErrInvalidMnemonic          = errors.New("invalid mnemonic")
	ErrInvalidMnemonicWordCount = errors.New("invalid mnemonic word count")
	ErrBytesLengthInvalid       = errors.New("bytes array must be multiple of 32")
	ErrInvalidChecksum          = errors.New("invalid checksum")
	ErrInvalidWord              = errors.New("invalid word")
)
