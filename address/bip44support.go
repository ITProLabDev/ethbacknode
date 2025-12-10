package address

import (
	"module github.com/ITProLabDev/ethbacknode/common/bip32"
	"module github.com/ITProLabDev/ethbacknode/common/bip39"
	"module github.com/ITProLabDev/ethbacknode/common/bip44"
	"strings"
)

const (
	DefaultEntropyBitLen = 128
)

func (p *Manager) GenerateBit44Address() (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = createNewBIP44Address(p.config.Bip36MnemonicLen, bip44CoinType, p.addressCodec)
	if err != nil {
		return nil, err
	}
	return addressRecord, nil
}

func (p *Manager) GenerateBit44AddressWithLen(mnemonicLen int) (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = createNewBIP44Address(mnemonicLen, bip44CoinType, p.addressCodec)
	if err != nil {
		return nil, err
	}
	return addressRecord, nil
}

func (p *Manager) RecoverBit44Address(mnemonic []string) (addressRecord *Address, err error) {

	var bip44CoinType = bip44.CoinType(p.config.Bip44CoinType)

	addressRecord, err = recoverBIP44AddressFromMnemonic(mnemonic, bip44CoinType, p.addressCodec)

	return addressRecord, nil
}

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

func recoverBIP44AddressFromMnemonic(mnemonic []string, coinType uint32, addressCodec AddressCodec) (addressRecord *Address, err error) {
	mnemonicStr := strings.Join(mnemonic, " ")
	entropy, err := bip39.EntropyFromMnemonic(mnemonicStr)
	if err != nil {
		return nil, err
	}
	return bip44EntropyToAddressRecord(entropy, coinType, addressCodec)
}
