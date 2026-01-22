package address

import (
	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/crypto"
)

// IsAddressKnown checks if an address is managed by this pool.
// Thread-safe operation.
func (p *Manager) IsAddressKnown(address string) bool {
	p.mux.RLock()
	_, found := p.fastPool.LookupString(address)
	p.mux.RUnlock()
	return found
}

// GetAddress retrieves an address record by address string.
// Returns ErrAddressUnknown if not found. Thread-safe.
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

// GetFreeAddressAndSubscribe retrieves a free address and subscribes it.
// Marks the address as subscribed with the given service/user/invoice IDs.
// Triggers pool refill if needed. Thread-safe.
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

// AddAddressRecordsBulk adds multiple address records in bulk.
// Skips addresses that already exist.
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

// AddAddressRecord adds a single address record to the pool.
// Returns ErrAddressExists if the address already exists.
func (p *Manager) AddAddressRecord(address *Address) (err error) {
	if p.IsAddressKnown(address.Address) {
		return ErrAddressExists
	}
	p.mux.Lock()
	err = p.addAddressUnsafe(address)
	p.mux.Unlock()
	return err
}

// AddAddressFill adds an address with custom initialization via fill function.
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

// AddPrivateKeyHex adds an address by its private key in hex format.
func (p *Manager) AddPrivateKeyHex(privateKeyHex string) (address string, err error) {
	pkBytes, err := hexnum.ParseHexBytes(privateKeyHex)
	if err != nil {
		return "", err
	}
	return p.AddPrivateKey(pkBytes)
}

// AddPrivateKey adds an address by its raw private key bytes.
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

// AddPrivateKeyHexFill adds an address by its hex-encoded private key with custom initialization.
// The fillParams function allows setting additional fields on the address record.
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

// WalkAllAddresses iterates over all addresses and calls walker for each.
// Thread-safe read operation.
func (p *Manager) WalkAllAddresses(walker func(address *Address)) {
	p.mux.RLock()
	for _, address := range p.allAddresses {
		walker(address)
	}
	p.mux.RUnlock()
}

// updateAddressUnsafe updates an address record using the provided updater function.
// Manages free address pool membership based on subscription status.
// UNSAFE: Caller must hold the mutex lock.
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

// addAddressUnsafe adds an address to the pool without locking.
// Adds to free addresses if not subscribed, triggers pool update.
// UNSAFE: Caller must hold the mutex lock.
func (p *Manager) addAddressUnsafe(address *Address) (err error) {
	p.allAddresses[address.Address] = address
	if !address.Subscribed {
		p.freeAddresses[address.Address] = address
	}
	go p.updatePool()
	return p.db.Save(address)
}

// getFreeAddressUnsafe returns any available free address from the pool.
// Returns ErrNoFreeAddresses if the pool is empty.
// UNSAFE: Caller must hold the mutex lock.
func (p *Manager) getFreeAddressUnsafe() (addressRecord *Address, err error) {
	for _, addressRecord = range p.freeAddresses {
		return addressRecord, nil
	}
	return nil, ErrNoFreeAddresses
}
