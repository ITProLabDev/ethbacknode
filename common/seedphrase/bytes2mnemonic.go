package seedphrase

import (
	"encoding/binary"
	"fmt"
)

func Bytes2Mnemonic(data []byte) ([]string, error) {
	dataBitLength := len(data) * 8
	checksumBitLength := dataBitLength / 32
	sentenceLength := (dataBitLength + checksumBitLength) / 11
	err := validateBytesBitSize(dataBitLength)
	if err != nil {
		return nil, err
	}
	mnemonic := make([]string, 0, sentenceLength)
	dataSignedBits := addChecksumBits(data)
	if len(dataSignedBits)%11 != 0 {
		return nil, ErrBytesLengthInvalid
	}
	var indexBits []uint8
	for i := 0; i < sentenceLength; i++ {
		indexBits, dataSignedBits = popBitsFromBitArray(dataSignedBits, 11)
		indexBits = bitArrayToBytes(padBitsArray(indexBits, 16))
		index := binary.BigEndian.Uint16(indexBits)
		fmt.Println(index, "-", words[index])
		mnemonic = append(mnemonic, words[index])
	}
	return mnemonic, nil
}
