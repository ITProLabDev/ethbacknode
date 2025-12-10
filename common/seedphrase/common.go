package seedphrase

import (
	"crypto/sha256"
	"fmt"
)

// validateBytesBitSize ensures that data bit size is the correct size for being a
// mnemonic.
func validateBytesBitSize(bitSize int) error {
	if (bitSize%32) != 0 || bitSize < 128 || bitSize > 256 {
		return ErrBytesLengthInvalid
	}
	return nil
}

// validateMnemonicWordCount ensures that data bit size is the correct mnemonic len.
func validateMnemonicWordCount(mnemonic []string) error {
	dataBitsLen := len(mnemonic) * 11
	payloadBitLen := dataBitsLen / 32 * 32
	crcBitLen := dataBitsLen - payloadBitLen
	if crcBitLen != payloadBitLen/32 {
		return ErrInvalidMnemonicWordCount
	}
	return nil
}
func extractDataFromBitsArray(bitsArray []uint8) []byte {
	dataBits, _ := shiftBitsFromBitArray(bitsArray, len(bitsArray)/32)
	return bitArrayToBytes(dataBits)
}
func uint16ToBitArray(index uint16) []uint8 {
	indexBits := make([]uint8, 16)
	for i := 0; i < 16; i++ {
		indexBits[15-i] = uint8(index & 1)
		index = index >> 1
	}
	return indexBits
}

// Add to data the first bits of the result of sha256(data)
func addChecksumBits(data []byte) []uint8 {
	// Get first byte of sha256
	hash := getSha256Hash(data)
	firstChecksumByte := hash[0]
	checksumBitLength := uint(len(data) / 4)
	dataBits := bytesToBitArray(data)
	checksumBits := bytesToBitArray([]byte{firstChecksumByte})
	if len(checksumBits) > int(checksumBitLength) {
		checksumBits = checksumBits[:int(checksumBitLength)]
	}
	dataSignedBits := addBitsToBitArray(dataBits, checksumBits)
	return dataSignedBits
}

// Validate checksum bits
func validateChecksumBits(dataBits []uint8) error {
	dataBitLength := len(dataBits)
	checksumBitLength := dataBitLength / 32
	fmt.Println("dataBitLength", dataBitLength, "checksumBitLength", checksumBitLength)
	dataBits, checksumBits := shiftBitsFromBitArray(dataBits, checksumBitLength)
	fmt.Println("checksumBits", checksumBits)
	if len(dataBits)%8 != 0 {
		return ErrInvalidChecksum
	}
	checksumByte := getSha256Hash(bitArrayToBytes(dataBits))[0]
	fmt.Println("checksumByte", fmt.Sprintf("%08b", checksumByte))
	calculatedChecksumBits := bytesToBitArray([]byte{checksumByte})[0:checksumBitLength]
	fmt.Println("calculatedChecksumBits", calculatedChecksumBits)
	for i, b := range checksumBits {
		if b != calculatedChecksumBits[i] {
			return ErrInvalidChecksum
		}
	}
	return nil
}
func getSha256Hash(data []byte) []byte {
	hasher := sha256.New()
	_, _ = hasher.Write(data)
	return hasher.Sum(nil)
}

// bytesToBitArray converts a byte slice to a bit array.
func bytesToBitArray(data []byte) []uint8 {
	var out []uint8
	for _, b := range data {
		for i := 0; i < 8; i++ {
			bit := (b >> uint(7-i)) & 1
			out = append(out, bit)
		}
	}
	return out
}

func bitArrayToBytes(bitArray []uint8) []byte {
	var out []byte
	for i := 0; i < len(bitArray); i += 8 {
		var b uint8
		for j := 0; j < 8; j++ {
			b = b << 1
			b = b | bitArray[i+j]
		}
		out = append(out, b)
	}
	return out
}

func addBitsToBitArray(bitArray []uint8, bits []uint8) []uint8 {
	return append(bitArray, bits...)
}

func pushBitsToBitArray(bitArray []uint8, bits []uint8) []uint8 {
	return append(bits, bitArray...)
}

func shiftBitsFromBitArray(bitArray []uint8, bitsCount int) ([]uint8, []uint8) {
	return bitArray[:len(bitArray)-bitsCount], bitArray[len(bitArray)-bitsCount:]
}
func popBitsFromBitArray(bitArray []uint8, bitsCount int) ([]uint8, []uint8) {
	return bitArray[0:bitsCount], bitArray[bitsCount:]
}

func padBitsArray(bitsArray []uint8, length int) []uint8 {
	offset := length - len(bitsArray)
	if offset <= 0 {
		return bitsArray
	}
	newBitsArray := make([]uint8, length)
	copy(newBitsArray[offset:], bitsArray)
	return newBitsArray
}
