package security

import "encoding/json"

func (m *Manager) SignParams(apiKey string, method string, params map[string]json.RawMessage) (sign string, err error) {
	panic("implement me")
}
