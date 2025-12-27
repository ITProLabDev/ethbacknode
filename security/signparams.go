package security

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"

	"github.com/ITProLabDev/ethbacknode/common/hexnum"
	"github.com/ITProLabDev/ethbacknode/tools/log"
	"golang.org/x/crypto/ripemd160"
)

func (m *Manager) SignRequest(apiKey string, method string, params json.RawMessage) (sign []byte, err error) {
	//todo decode api key
	var keyBytes []byte
	switch m.config.KeyFormat {
	case KEY_FORMAT_HEX:
		keyBytes, err = hexnum.ParseHexBytes(apiKey)
		if err != nil {
			return nil, err
		}
	case KEY_FORMAT_BASE58:
		//todo
	case KEY_FORMAT_JSON:
		//todo
	}

	requestBytes := append(keyBytes, append([]byte(method), params...)...)

	signBytes := m._signBytes(requestBytes, m.config.SignatureType)

	return signBytes, nil
}

func (m *Manager) _signBytes(data []byte, signType string) []byte {
	switch signType {
	case SIGNATURE_TYPE_SHA256:
		return _hashBytesSha256(data)
	case SIGNATURE_TYPE_SHA512:
		return _hashBytesSha512(data)
	case SIGNATURE_TYPE_RIPEMD:
		return _hashBytesRIPEMD(data)
	default:
		log.Warning("unsupported signType:", signType, "use default")
		return m._signBytes(data, m.config.SignatureType)
	}
}

func _hashBytesSha256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func _hashBytesSha512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

func _hashBytesRIPEMD(data []byte) []byte {
	hash := ripemd160.New()
	hash.Write(data)
	return hash.Sum(nil)
}
