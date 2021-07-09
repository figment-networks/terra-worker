package types

// GetBlockchainResponseV4 cosmos response from blockchain
type GetBlockchainResponseV4 struct {
	ID     int                `json:"id"`
	RPC    string             `json:"jsonrpc"`
	Result ResultBlockchainV4 `json:"result"`
	Error  Error              `json:"error"`
}

// ResultBlockchainV4 is result of fetching block
type ResultBlockchainV4 struct {
	LastHeight string        `json:"last_height"`
	BlockMetas []BlockMetaV4 `json:"block_metas"`
}

// BlockMetaV4 is block metadata
type BlockMetaV4 struct {
	BlockID BlockID       `json:"block_id"`
	Header  BlockHeaderV4 `json:"header"`
	NumTxs  string        `json:"num_txs"`
}

// BlockMetaV4 is block header
type BlockHeaderV4 struct {
	Height  string `json:"height"`
	ChainID string `json:"chain_id"`
	Time    string `json:"time"`
}

// GetBlockResponse terra response from /block

type GetBlockResponseV4 struct {
	ID     string        `json:"id"`
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
