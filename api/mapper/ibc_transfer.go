package mapper

import (
	"fmt"
	"strconv"

	"github.com/figment-networks/indexing-engine/structs"
	shared "github.com/figment-networks/indexing-engine/structs"

	transfer "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	"github.com/gogo/protobuf/proto"
)

// IBCTransferToSub transforms ibc.MsgTransfer sdk messages to SubsetEvent
func IBCTransferToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &transfer.MsgTransfer{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a transfer type: %w", err)
	}

	amount := structs.TransactionAmount{
		Currency: m.Token.Denom,
		Numeric:  m.Token.Amount.BigInt(),
		Text:     m.Token.Amount.String(),
	}

	return shared.SubsetEvent{
		Type:   []string{"transfer"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"sender":   {{ID: m.Sender}},
			"receiver": {{ID: m.Receiver}},
		},
		Sender: []structs.EventTransfer{
			{
				Account: structs.Account{ID: m.Sender},
				Amounts: []structs.TransactionAmount{amount},
			},
		},
		Recipient: []structs.EventTransfer{
			{
				Account: structs.Account{ID: m.Receiver},
				Amounts: []structs.TransactionAmount{amount},
			},
		},
		Additional: map[string][]string{
			"source_port":                    {m.SourcePort},
			"source_channel":                 {m.SourceChannel},
			"timeout_height_revision_number": {strconv.FormatUint(m.TimeoutHeight.RevisionNumber, 10)},
			"timeout_height_revision_height": {strconv.FormatUint(m.TimeoutHeight.RevisionHeight, 10)},
			"timeout_stamp":                  {strconv.FormatUint(m.TimeoutTimestamp, 10)},
		},
	}, nil
}
