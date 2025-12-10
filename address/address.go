package address

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"module github.com/ITProLabDev/ethbacknode/crypto"
	"module github.com/ITProLabDev/ethbacknode/crypto/secp256k1"
)

type Address struct {
	Address      string `json:"address" storm:"id"`
	AddressBytes []byte `json:"addressBytes" storm:"index"`
	PrivateKey   []byte `json:"privateKey"`
	Master       bool   `json:"master"`

	Subscribed bool  `json:"subscribed"`
	ServiceId  int   `json:"serviceId"`
	UserId     int64 `json:"userId"`
	InvoiceId  int64 `json:"invoiceId"`
	WatchOnly  bool  `json:"watchOnly"`

	Bip39Support  bool     `json:"bip39Support,omitempty"`
	Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
}

type AddressCodec interface {
	EncodeBytesToAddress(addressBytes []byte) (string, error)
	DecodeAddressToBytes(address string) ([]byte, error)
	PrivateKeyToAddress(privateKey []byte) (string, []byte, error)
	IsValid(address string) bool
}

func (a *Address) String() string {
	return a.Address
}

func (a *Address) GetKey() []byte {
	return a.AddressBytes
}

func (m *Address) Encode() []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	enc.Encode(m)
	return b.Bytes()
}

func (m *Address) Decode(b []byte) error {
	buf := bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)
	return enc.Decode(m)
}

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

func generatePrivateKey() (privateKey []byte, err error) {
	ecdsaKey, err := ecdsa.GenerateKey(secp256k1.P256k1(), rand.Reader)
	if err != nil {
		return nil, err
	}
	privateKey = crypto.BytesFromECDSAPrivateKey(ecdsaKey)
	return privateKey, nil
}
