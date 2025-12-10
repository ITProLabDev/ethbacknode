package uniclient

type AddressInfo struct {
	Address       string   `json:"address"`
	PrivateKey    string   `json:"privateKey,omitempty"`
	UserId        int64    `json:"userId,omitempty"`
	InvoiceId     int64    `json:"invoiceId,omitempty"`
	WatchOnly     bool     `json:"watchOnly,omitempty"`
	Bip39Support  bool     `json:"bip39Support,omitempty"`
	Bip39Mnemonic []string `json:"bip39Mnemonic,omitempty"`
}

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
