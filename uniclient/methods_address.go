package uniclient

// AddressInfo contains information about a managed address.
type AddressInfo struct {
	Address       string   `json:"address"`
	PrivateKey    string   `json:"privateKey,omitempty"`
	UserId        int64    `json:"userId,omitempty"`
	InvoiceId     int64    `json:"invoiceId,omitempty"`
	WatchOnly     bool     `json:"watchOnly,omitempty"`
	Bip39Support  bool     `json:"bip39Support,omitempty"`
	Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
}

// AddressGetNew generates a new address for the given user and invoice.
func (c *Client) AddressGetNew(userId, invoiceId int64, watchOnly bool) (addressInfo *AddressInfo, err error) {
	type addressGetNewRequest struct {
		ServiceId int   `json:"serviceId"`
		UserId    int64 `json:"userId"`
		InvoiceId int64 `json:"invoiceId"`
		WatchOnly bool  `json:"watchOnly"`
		FullInfo  bool  `json:"fullInfo"`
	}
	type addressGetNewResponse struct {
		Address       string   `json:"address"`
		PrivateKey    string   `json:"privateKey,omitempty"`
		UserId        int64    `json:"userId,omitempty"`
		InvoiceId     int64    `json:"invoiceId,omitempty"`
		WatchOnly     bool     `json:"watchOnly,omitempty"`
		Bip39Support  bool     `json:"bip39Support,omitempty"`
		Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
	}
	request := NewRequest("addressGetNew", &addressGetNewRequest{
		ServiceId: c.serviceId,
		UserId:    userId,
		InvoiceId: invoiceId,
		WatchOnly: watchOnly,
	})
	addrResponse := new(addressGetNewResponse)
	err = c.rpcCall(request, addrResponse)
	if err != nil {
		return nil, err
	}
	return &AddressInfo{
		Address:   addrResponse.Address,
		UserId:    addrResponse.UserId,
		InvoiceId: addrResponse.InvoiceId,
		WatchOnly: addrResponse.WatchOnly,
	}, nil
}

// AddressGetNewFullInfo generates a new address and returns full info including private key.
func (c *Client) AddressGetNewFullInfo(userId, invoiceId int64, watchOnly bool) (addressInfo *AddressInfo, err error) {
	type addressGetNewRequest struct {
		ServiceId int   `json:"serviceId"`
		UserId    int64 `json:"userId"`
		InvoiceId int64 `json:"invoiceId"`
		WatchOnly bool  `json:"watchOnly"`
		FullInfo  bool  `json:"fullInfo"`
	}
	type addressGetNewResponse struct {
		Address       string   `json:"address"`
		PrivateKey    string   `json:"privateKey,omitempty"`
		UserId        int64    `json:"userId,omitempty"`
		InvoiceId     int64    `json:"invoiceId,omitempty"`
		WatchOnly     bool     `json:"watchOnly,omitempty"`
		Bip39Support  bool     `json:"bip39Support,omitempty"`
		Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
	}
	request := NewRequest("addressGetNew", &addressGetNewRequest{
		ServiceId: c.serviceId,
		UserId:    userId,
		InvoiceId: invoiceId,
		WatchOnly: watchOnly,
		FullInfo:  true,
	})
	addrResponse := new(addressGetNewResponse)
	err = c.rpcCall(request, addrResponse)
	if err != nil {
		return nil, err
	}
	return &AddressInfo{
		Address:       addrResponse.Address,
		UserId:        addrResponse.UserId,
		InvoiceId:     addrResponse.InvoiceId,
		WatchOnly:     addrResponse.WatchOnly,
		PrivateKey:    addrResponse.PrivateKey,
		Bip39Support:  addrResponse.Bip39Support,
		Bip39Mnemonic: addrResponse.Bip39Mnemonic,
	}, nil
}

// AddressSubscribeResult is the result of AddressSubscribe. Failures are
// reported here (Success false, Error set), not as a Go error -- the server
// answers this call the same way.
type AddressSubscribeResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// AddressSubscribe registers an address as managed by the client's
// configured service, so its transfers are tracked. Exactly one of addr,
// privateKeyHex, or mnemonic identifies the address:
//   - addr alone subscribes an existing address watch-only.
//   - privateKeyHex (0x-hex) subscribes the address it derives, held by this
//     node.
//   - mnemonic (BIP-39 words) recovers and subscribes the address it derives.
//
// watchOnly marks the address as tracked without this node holding funds for
// it; it is implied true whenever no private key or mnemonic is given.
func (c *Client) AddressSubscribe(addr, privateKeyHex string, mnemonic []string, userId, invoiceId int64, watchOnly bool) (result *AddressSubscribeResult, err error) {
	type addressSubscribeRequest struct {
		Address    string   `json:"address,omitempty"`
		PrivateKey string   `json:"privateKey,omitempty"`
		Mnemonic   []string `json:"mnemonic,omitempty"`
		ServiceID  int      `json:"serviceId"`
		UserId     int64    `json:"userId,omitempty"`
		InvoiceId  int64    `json:"invoiceId,omitempty"`
		WatchOnly  bool     `json:"watchOnly,omitempty"`
	}
	request := NewRequest("addressSubscribe", &addressSubscribeRequest{
		Address: addr, PrivateKey: privateKeyHex, Mnemonic: mnemonic,
		ServiceID: c.serviceId, UserId: userId, InvoiceId: invoiceId, WatchOnly: watchOnly,
	})
	result = new(AddressSubscribeResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// MnemonicAddressResult is the result of AddressRecover and AddressGenerate:
// both answer with the same shape (an address, its private key, and the
// mnemonic behind it), and report failure the same way AddressSubscribe does
// -- Success false with Error set, not a Go error.
type MnemonicAddressResult struct {
	Success       bool     `json:"success"`
	Address       string   `json:"address,omitempty"`
	PrivateKey    string   `json:"privateKey,omitempty"`
	Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// AddressRecover restores an address (and its private key) from a BIP-39
// mnemonic. Open (no auth required) -- the mnemonic itself is the secret.
func (c *Client) AddressRecover(mnemonic []string) (result *MnemonicAddressResult, err error) {
	type addressRecoverRequest struct {
		Mnemonic []string `json:"mnemonic"`
	}
	request := NewRequest("addressRecover", &addressRecoverRequest{Mnemonic: mnemonic})
	result = new(MnemonicAddressResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// AddressGenerate creates a brand-new address from a freshly generated
// mnemonic, without subscribing it to any service. mnemonicLen is the number
// of BIP-39 words (0 defaults to 12, matching the server's own default).
// Open (no auth required).
func (c *Client) AddressGenerate(mnemonicLen int) (result *MnemonicAddressResult, err error) {
	if mnemonicLen == 0 {
		mnemonicLen = 12
	}
	type addressGenerateRequest struct {
		MnemonicLen int `json:"mnemonicLen"`
	}
	request := NewRequest("addressGenerate", &addressGenerateRequest{MnemonicLen: mnemonicLen})
	result = new(MnemonicAddressResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}
