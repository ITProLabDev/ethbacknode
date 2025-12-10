package uniclient

type TokenInfo struct {
	ContractAddress string `json:"contractAddress,omitempty"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Protocol        string `json:"protocol,omitempty"`
}

type NodeInfo struct {
	Blockchain string       `json:"blockchain"`
	Id         string       `json:"id"`
	Symbol     string       `json:"symbol"`
	Decimals   int          `json:"decimals"`
	Protocols  []string     `json:"protocols"`
	Tokens     []*TokenInfo `json:"tokens"`
}

func (c *Client) GetNodeInfo() (nodeInfo *NodeInfo, err error) {
	request := NewRequest("getNodeInfo", nil)
	nodeInfo = new(NodeInfo)
	err = c.rpcCall(request, nodeInfo)
	if err != nil {
		return nil, err
	}
	return nodeInfo, nil
}
