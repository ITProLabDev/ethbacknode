package uniclient

// PingResult is the result of Ping.
type PingResult struct {
	Result    string `json:"result"`
	Timestamp int64  `json:"timestamp"`
}

// Ping checks connectivity and service availability. Open (no auth required).
func (c *Client) Ping() (result *PingResult, err error) {
	request := NewRequest("ping", nil)
	result = new(PingResult)
	if err = c.rpcCall(request, result); err != nil {
		return nil, err
	}
	return result, nil
}

// TokenListEntry is one row of InfoGetTokenList: the native coin (Token
// false, ContractAddress empty) or a supported token (Token true).
type TokenListEntry struct {
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Token           bool   `json:"token,omitempty"`
	ContractAddress string `json:"contractAddress"`
}

// InfoGetTokenList returns the native coin and every supported token. Open
// (no auth required).
func (c *Client) InfoGetTokenList() (tokens []*TokenListEntry, err error) {
	request := NewRequest("infoGetTokenList", nil)
	if err = c.rpcCall(request, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// TokenInfo describes an ERC-20 or other token contract.
type TokenInfo struct {
	ContractAddress string `json:"contractAddress,omitempty"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Protocol        string `json:"protocol,omitempty"`
}

// NodeInfo describes the EthBackNode server and supported tokens.
type NodeInfo struct {
	Blockchain string       `json:"blockchain"`
	Id         string       `json:"id"`
	Symbol     string       `json:"symbol"`
	Decimals   int          `json:"decimals"`
	Protocols  []string     `json:"protocols"`
	Tokens     []*TokenInfo `json:"tokens"`
}

// GetNodeInfo retrieves information about the node, including supported tokens.
func (c *Client) GetNodeInfo() (nodeInfo *NodeInfo, err error) {
	request := NewRequest("getNodeInfo", nil)
	nodeInfo = new(NodeInfo)
	err = c.rpcCall(request, nodeInfo)
	if err != nil {
		return nil, err
	}
	return nodeInfo, nil
}
