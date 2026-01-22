package address

import (
	"github.com/ITProLabDev/ethbacknode/common/bip32"
	"github.com/ITProLabDev/ethbacknode/common/bip39"
	"github.com/ITProLabDev/ethbacknode/common/bip44"
	"strings"
)

// DefaultEntropyBitLen is the default entropy length (128 bits = 12 word mnemonic).
const (
	DefaultEntropyBitLen = 128
)

// GenerateBit44Address creates a new BIP-44 HD wallet address using the configured
// mnemonic length and coin type. Returns an address with BIP-39 mnemonic.
func (p *Manager) GenerateBit44Address() (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = createNewBIP44Address(p.config.Bip36MnemonicLen, bip44CoinType, p.addressCodec)
	if err != nil {
		return nil, err
	}
	return addressRecord, nil
}

// GenerateBit44AddressWithLen creates a new BIP-44 HD wallet address with a custom
// mnemonic length (12 or 24 words).
func (p *Manager) GenerateBit44AddressWithLen(mnemonicLen int) (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = createNewBIP44Address(mnemonicLen, bip44CoinType, p.addressCodec)
	if err != nil {
		return nil, err
	}
	return addressRecord, nil
}

// RecoverBit44Address recovers a BIP-44 HD wallet address from an existing mnemonic phrase.
// Returns the address derived from the mnemonic.
func (p *Manager) RecoverBit44Address(mnemonic []string) (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = recoverBIP44AddressFromMnemonic(mnemonic, bip44CoinType, p.addressCodec)

	return addressRecord, nil
}

// bip44EntropyToAddressRecord derives a BIP-44 address from entropy bytes.
// Creates mnemonic, derives master key, and generates address at path m/44'/coinType'/0'/0/0.
func bip44EntropyToAddressRecord(entropy []byte, coinType uint32, addressCodec AddressCodec) (addressRecord *Address, err error) {
	mnemonicStr, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}
	mnemonic := strings.Split(mnemonicStr, " ")
	seed := bip39.NewSeed(mnemonicStr, "")
	masterKey, err := bip32.NewMasterKey(seed)
	key, err := bip44.NewKeyFromMasterKey(masterKey, coinType, 0x80000000, 0, 0)
	if err != nil {
		return nil, err
	}
	privateKey := key.Key
	addressStr, addressBytes, err := addressCodec.PrivateKeyToAddress(privateKey)
	if err != nil {
		return nil, err
	}
	addressRecord = &Address{
		Address:       addressStr,
		AddressBytes:  addressBytes,
		PrivateKey:    privateKey,
		Bip39Support:  true,
		Bip39Mnemonic: mnemonic,
	}
	return addressRecord, nil
}

// createNewBIP44Address generates a new BIP-44 address with random entropy.
// Supports 12-word (128-bit) or 24-word (256-bit) mnemonics.
func createNewBIP44Address(mnemonicLen int, coinType uint32, addressCodec AddressCodec) (addressRecord *Address, err error) {
	bitLen := DefaultEntropyBitLen
	if mnemonicLen == 12 {
		bitLen = 128
	} else if mnemonicLen == 24 {
		bitLen = 256
	} else {
		return nil, ErrInvalidMnemonicLen
	}
	entropy, err := bip39.NewEntropy(bitLen)
	if err != nil {
		return nil, err
	}
	return bip44EntropyToAddressRecord(entropy, coinType, addressCodec)
}

// recoverBIP44AddressFromMnemonic recovers a BIP-44 address from a mnemonic word list.
// Converts mnemonic to entropy and derives the address.
func recoverBIP44AddressFromMnemonic(mnemonic []string, coinType uint32, addressCodec AddressCodec) (addressRecord *Address, err error) {
	mnemonicStr := strings.Join(mnemonic, " ")
	entropy, err := bip39.EntropyFromMnemonic(mnemonicStr)
	if err != nil {
		return nil, err
	}
	return bip44EntropyToAddressRecord(entropy, coinType, addressCodec)
}
