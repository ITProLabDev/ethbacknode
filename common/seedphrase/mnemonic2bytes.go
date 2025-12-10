package seedphrase

func Mnemonic2Bytes(mnemonic []string) (data []byte, err error) {
	if err = validateMnemonicWordCount(mnemonic); err != nil {
		return nil, err
	}
	bitsArray := make([]uint8, 0, len(mnemonic)*11)
	for _, word := range mnemonic {
		index, ok := wordsIndex[word]
		if !ok {
			return nil, ErrInvalidWord
		}
		indexBits := uint16ToBitArray(index)[5:]
		bitsArray = addBitsToBitArray(bitsArray, indexBits)
	}
	if err = validateChecksumBits(bitsArray); err != nil {
		return nil, err
	}
	data = extractDataFromBitsArray(bitsArray)
	return data, nil
}
