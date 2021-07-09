package types

// GetBlockResponse terra response from /block

type GetBlockResponseV4 struct {
	// ID     string        `json:"id"`
	RPC    string        `json:"jsonrpc"`
	Result ResultBlockV4 `json:"result"`
	Error  Error         `json:"error"`
}

type ResultBlockV4 struct {
	Block   BlockV4 `json:"block"`
	BlockID BlockID `json:"block_id"`
}

// BlockV4 is terra block data
type BlockV4 struct {
	Header BlockHeaderV4 `json:"header"`
	Data   BlockDataV4   `json:"data"`
}

type BlockDataV4 struct {
	Txs []string `json:"txs"`
}

// BlockHeaderV4 is block header
type BlockHeaderV4 struct {
	Height  string `json:"height"`
	ChainID string `json:"chain_id"`
	Time    string `json:"time"`
}
