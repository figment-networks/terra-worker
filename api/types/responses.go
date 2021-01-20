package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TxResponse is result of querying for a tx
type TxResponse struct {
	Hash     string            `json:"hash"`
	Height   string            `json:"height"`
	TxResult ResponseDeliverTx `json:"tx_result"`

	// TxData is base64 encoded transaction data
	TxData string `json:"tx"`
}

type ResponseDeliverTx struct {
	Log string `json:"log"`

	GasWanted string     `json:"gasWanted"`
	GasUsed   string     `json:"gasUsed"`
	Events    []TxEvents `json:"tags"`
}

type TxTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RewardResponse is terra response for querying /rewards
type RewardResponse struct {
	Height string       `json:"height"`
	Result RewardResult `json:"result"`
}
type RewardResult struct {
	Total sdk.DecCoins `json:"total"`
}

type BlockHeader struct {
	Height  string `json:"height"`
	ChainID string `json:"chain_id"`
	Time    string `json:"time"`
	NumTxs  string `json:"num_txs"`
}

// ResultBlockchain is result of fetching block
type ResultBlockchain struct {
	LastHeight string      `json:"last_height"`
	BlockMetas []BlockMeta `json:"block_metas"`
}

// BlockMeta is block metadata
type BlockMeta struct {
	BlockID BlockID     `json:"block_id"`
	Header  BlockHeader `json:"header"`
}

// BlockID info
type BlockID struct {
	Hash string `json:"hash"`
}

// Block is cosmos block data
type Block struct {
	Header BlockHeader `json:"header"`
}

type Error struct {
	Code      int    `json:"code"`
	CodeSpace string `json:"codespace"`
	Message   string `json:"message"`
	Data      string `json:"data"`
}

// Result of searching for txs
type ResultTxSearch struct {
	Txs        []TxResponse `json:"txs,omitempty"`
	TotalCount string       `json:"total_count,omitempty"`
	Error      Error        `json:"error"`
}

// GetTxSearchResponse cosmos response for search
type GetTxSearchResponse struct {
	//	ID     string         `json:"id"`
	RPC    string         `json:"jsonrpc"`
	Result ResultTxSearch `json:"result"`
	Error  Error          `json:"error"`
}

// GetBlockchainResponse cosmos response from blockchain
type GetBlockchainResponse struct {
	ID     string           `json:"id"`
	RPC    string           `json:"jsonrpc"`
	Result ResultBlockchain `json:"result"`
	Error  Error            `json:"error"`
}
