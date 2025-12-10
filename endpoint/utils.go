package endpoint

func (r *BackRpc) addressNormalise(address string) (string, error) {
	addressBytes, err := r.addressCodec.DecodeAddressToBytes(address)
	if err != nil {
		return "", err
	}
	address, _ = r.addressCodec.EncodeBytesToAddress(addressBytes)
	return address, nil
}
