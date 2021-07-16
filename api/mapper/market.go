package mapper

import (
	"fmt"

	"github.com/figment-networks/indexing-engine/structs"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	market "github.com/terra-money/core/x/market/types"
)

func MarketSwapToSub(msg []byte, lg types.ABCIMessageLog) (se structs.SubsetEvent, err error) {
	swap := &market.MsgSwap{}
	if err := proto.Unmarshal(msg, swap); err != nil {
		return se, fmt.Errorf("Not a swap type: %w", err)
	}

	offerRt := structs.TransactionAmount{
		Currency: swap.OfferCoin.Denom,
		Text:     swap.OfferCoin.Amount.String(),
		Numeric:  swap.OfferCoin.Amount.BigInt(),
	}
	ask := structs.TransactionAmount{Currency: swap.AskDenom}

	se = structs.SubsetEvent{
		Type:   []string{"swap"},
		Module: "market",
		Node: map[string][]structs.Account{
			"trader": {{ID: swap.Trader}},
		},
		Sender: []structs.EventTransfer{
			{
				Account: structs.Account{ID: swap.Trader},
				Amounts: []structs.TransactionAmount{offerRt, ask},
			},
		},
		Amount: map[string]structs.TransactionAmount{
			"offer": offerRt,
			"ask":   ask,
		},
	}

	err = produceTransfers(&se, "send", "", lg)
	return se, err
}

func MarketSwapSendToSub(msg []byte, lg types.ABCIMessageLog) (se structs.SubsetEvent, err error) {
	swap := &market.MsgSwapSend{}
	if err := proto.Unmarshal(msg, swap); err != nil {
		return se, fmt.Errorf("Not a swapsend type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"swapsend"},
		Module: "market",
	}

	offerRt := structs.TransactionAmount{
		Currency: swap.OfferCoin.Denom,
		Text:     swap.OfferCoin.Amount.String(),
		Numeric:  swap.OfferCoin.Amount.BigInt(),
	}
	ask := structs.TransactionAmount{Currency: swap.AskDenom}

	// se.Node = map[string][]structs.Account{}

	se.Sender = append(se.Sender, structs.EventTransfer{
		Account: structs.Account{ID: swap.FromAddress},
		Amounts: []structs.TransactionAmount{offerRt, ask},
	})

	se.Recipient = append(se.Sender, structs.EventTransfer{
		Account: structs.Account{ID: swap.ToAddress},
		Amounts: []structs.TransactionAmount{offerRt, ask},
	})

	se.Amount = map[string]structs.TransactionAmount{
		"offer": offerRt,
		"ask":   ask,
	}

	err = produceTransfers(&se, "send", "", lg)
	return se, err
}
