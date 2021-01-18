package api

import (
	"bytes"
	"encoding/json"

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

type LogFormatLog struct {
	Error
}

type LogFormat struct {
	MsgIndex float64      `json:"msg_index,omitempty"`
	Success  bool         `json:"success,omitempty"`
	Log      LogFormatLog `json:"log,omitempty"`
	Events   []TxEvents   `json:"events,omitempty"`
}

type logFormat struct {
	MsgIndex float64    `json:"msg_index,omitempty"`
	Success  bool       `json:"success,omitempty"`
	Log      string     `json:"log,omitempty"`
	Events   []TxEvents `json:"events,omitempty"`
}

func (lf *LogFormat) UnmarshalJSON(b []byte) error {
	llf := &logFormat{}

	if err := json.Unmarshal(b, llf); err != nil {
		return err
	}

	lf.MsgIndex = llf.MsgIndex
	lf.Success = llf.Success
	lf.Events = llf.Events
	if llf.Log != "" {
		return json.Unmarshal([]byte(llf.Log), &lf.Log)
	}
	return nil
}

type TxEvents struct {
	Type string `json:"type,omitempty"`
	//Attributes []string `json:"attributes"`
	Attributes *TxEventsAttributes `json:"attributes,omitempty"`
}

type TxEventsAttributes struct {
	Module    string
	Action    string
	Amount    []string
	Sender    []string
	Validator map[string][]string
	Withdraw  map[string][]string
	Recipient []string
	Voter     []string
	Feeder    []string

	CompletionTime string
	Commission     []string

	Denom []string

	Others map[string][]string
}

type kvHolder struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UnmarshalJSON LogEvents into a different format,
// to be able to parse it later more easily
// thats fulfillment of json.Unmarshaler inferface
func (lea *TxEventsAttributes) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	kc := &kvHolder{}

	// read open bracket
	_, err := dec.Token()
	if err != nil {
		return err
	}

	for dec.More() {
		err := dec.Decode(kc)
		if err != nil {
			return err
		}

		switch kc.Key {
		case "validator", "destination_validator", "source_validator":
			if lea.Validator == nil {
				lea.Validator = map[string][]string{}
			}
			v, ok := lea.Validator[kc.Key]
			if !ok {
				v = []string{}
			}
			lea.Validator[kc.Key] = append(v, kc.Value)
		case "sender":
			lea.Sender = append(lea.Sender, kc.Value)
		case "recipient":
			lea.Recipient = append(lea.Recipient, kc.Value)
		case "feeder":
			lea.Feeder = append(lea.Feeder, kc.Value)
		case "voter":
			lea.Voter = append(lea.Voter, kc.Value)
		case "module":
			lea.Module = kc.Value
		case "action":
			lea.Action = kc.Value
		case "completion_time":
			lea.CompletionTime = kc.Value
		case "amount":
			lea.Amount = append(lea.Amount, kc.Value)
		default:
			if lea.Others == nil {
				lea.Others = map[string][]string{}
			}

			k, ok := lea.Others[kc.Key]
			if !ok {
				k = []string{}
			}
			lea.Others[kc.Key] = append(k, kc.Value)
		}
	}
	// read closing bracket
	_, err = dec.Token()
	if err != nil {
		return err
	}

	return nil
}
