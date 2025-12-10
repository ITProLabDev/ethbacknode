package address

import (
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/crypto"
)

func (p *Manager) IsAddressKnown(address string) bool {
	p.mux.RLock()
	_, found := p.fastPool.LookupString(address)
	p.mux.RUnlock()
	return found
}

func (p *Manager) GetAddress(address string) (addressRecord *Address, err error) {
	var found bool
	p.mux.RLock()
	addressRecord, found = p.fastPool.LookupString(address)
	p.mux.RUnlock()
	if !found {
		return nil, ErrAddressUnknown
	}
	return addressRecord, nil
}

func (p *Manager) GetFreeAddressAndSubscribe(serviceId int, userId, invoiceId int64, watchOnly bool) (addressRecord *Address, err error) {
	p.mux.Lock()
	defer func() {
		p.mux.Unlock()
		go p.checkFreeAddressPool()
	}()
	addressRecord, err = p.getFreeAddressUnsafe()
	if err != nil {
		return nil, err
	}
	err = p.updateAddressUnsafe(addressRecord.Address, func(address *Address) error {
		address.ServiceId = serviceId
		address.UserId = userId
		address.InvoiceId = invoiceId
		address.WatchOnly = watchOnly
		address.Subscribed = true
		return nil
	})
	return addressRecord, nil
}

func (p *Manager) AddAddressRecordsBulk(addresses []*Address) (err error) {
	for _, address := range addresses {
		if !p.IsAddressKnown(address.Address) {
			p.mux.Lock()
			err = p.addAddressUnsafe(address)
			p.mux.Unlock()
			if err != nil {
				//todo add address info to error
				return err
			}
		}
	}
	p.mux.Lock()
	for _, address := range addresses {
		p.allAddresses[address.Address] = address
	}
	go p.updatePool()
	p.mux.Unlock()
	return nil
}

func (p *Manager) AddAddressRecord(address *Address) (err error) {
	if p.IsAddressKnown(address.Address) {
		return ErrAddressExists
	}
	p.mux.Lock()
	err = p.addAddressUnsafe(address)
	p.mux.Unlock()
	return err
}

func (p *Manager) AddAddressFill(addressString string, fill func(a *Address)) (addressRecord *Address, err error) {
	if p.IsAddressKnown(addressString) {
		return nil, ErrAddressExists
	}
	record, err := p.NewAddressRecordFill(addressString, fill)
	if err != nil {
		return nil, err
	}
	err = p.AddAddressRecord(record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (p *Manager) AddPrivateKeyHex(privateKeyHex string) (address string, err error) {
	pkBytes, err := hexnum.ParseHexBytes(privateKeyHex)
	if err != nil {
		return "", err
	}
	return p.AddPrivateKey(pkBytes)
}

func (p *Manager) AddPrivateKey(privateKeyBytes []byte) (address string, err error) {
	privateKey, _ := crypto.ECDSAKeysFromPrivateKeyBytes(privateKeyBytes)
	addressString, addressBytes, err := p.addressCodec.PrivateKeyToAddress(privateKeyBytes)
	if err != nil {
		return "", err
	}
	if p.IsAddressKnown(addressString) {
		return addressString, ErrAddressExists
	}
	newAddressRecord := &Address{
		Address:      addressString,
		AddressBytes: addressBytes,
		PrivateKey:   crypto.BytesFromECDSAPrivateKey(privateKey),
	}
	p.mux.Lock()
	err = p.addAddressUnsafe(newAddressRecord)
	p.mux.Unlock()
	return addressString, nil
}

func (p *Manager) AddPrivateKeyHexFill(privateKeyHex string, fillParams func(address *Address)) (addressString string, err error) {
	privateKeyBytes, err := hexnum.ParseHexBytes(privateKeyHex)
	if err != nil {
		return "", err
	}
	privateKey, _ := crypto.ECDSAKeysFromPrivateKeyBytes(privateKeyBytes)
	addressString, addressBytes, err := p.addressCodec.PrivateKeyToAddress(privateKeyBytes)
	if err != nil {
		return "", err
	}
	if p.IsAddressKnown(addressString) {
		return addressString, ErrAddressExists
	}
	newAddressRecord := &Address{
		Address:      addressString,
		AddressBytes: addressBytes,
		PrivateKey:   crypto.BytesFromECDSAPrivateKey(privateKey),
	}
	fillParams(newAddressRecord)
	p.mux.Lock()
	err = p.addAddressUnsafe(newAddressRecord)
	p.mux.Unlock()
	if err != nil {
		return "", err
	}
	return addressString, nil
}

func (p *Manager) WalkAllAddresses(walker func(address *Address)) {
	p.mux.RLock()
	for _, address := range p.allAddresses {
		walker(address)
	}
	p.mux.RUnlock()
}

func (p *Manager) updateAddressUnsafe(addressStr string, updater func(address *Address) error) (err error) {
	addressRecord, found := p.fastPool.LookupString(addressStr)
	if !found {
		return ErrAddressUnknown
	}
	err = updater(addressRecord)
	if err != nil {
		return err
	}
	if !addressRecord.Subscribed {
		p.freeAddresses[addressRecord.Address] = addressRecord
	} else {
		delete(p.freeAddresses, addressRecord.Address)
	}
	err = p.db.Save(addressRecord)
	if err != nil {
		return err
	}
	return nil
}

func (p *Manager) addAddressUnsafe(address *Address) (err error) {
	p.allAddresses[address.Address] = address
	if !address.Subscribed {
		p.freeAddresses[address.Address] = address
	}
	go p.updatePool()
	return p.db.Save(address)
}

func (p *Manager) getFreeAddressUnsafe() (addressRecord *Address, err error) {
	for _, addressRecord = range p.freeAddresses {
		return addressRecord, nil
	}
	return nil, ErrNoFreeAddresses
}
