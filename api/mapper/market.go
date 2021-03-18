package mapper

import (
	"errors"

	"github.com/figment-networks/indexer-manager/structs"
	"github.com/figment-networks/terra-worker/api/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"

	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/market"
)

func MarketSwapToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	swap, ok := msg.(market.MsgSwap)
	if !ok {
		return se, errors.New("Not a swap type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"swap"},
		Module: "market",
	}

	se.Node = map[string][]structs.Account{}
	if !swap.Trader.Empty() {
		traderBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, swap.Trader.Bytes())
		traderAccount := structs.Account{ID: traderBech32Addr}
		se.Node["trader"] = []structs.Account{traderAccount}

		offerRt := structs.TransactionAmount{
			Currency: swap.OfferCoin.Denom,
			Text:     swap.OfferCoin.Amount.String(),
			Numeric:  swap.OfferCoin.Amount.BigInt(),
		}
		ask := structs.TransactionAmount{Currency: swap.AskDenom}
		se.Sender = append(se.Sender, structs.EventTransfer{
			Account: traderAccount,
			Amounts: []structs.TransactionAmount{offerRt, ask},
		})

		se.Amount = map[string]structs.TransactionAmount{
			"offer": offerRt,
			"ask":   ask,
		}
	}

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}

func MarketSwapSendToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	swap, ok := msg.(market.MsgSwapSend)
	if !ok {
		return se, errors.New("Not a swapsend type")
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

	se.Node = map[string][]structs.Account{}

	fromBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, swap.FromAddress.Bytes())
	fromAccount := structs.Account{ID: fromBech32Addr}

	se.Sender = append(se.Sender, structs.EventTransfer{
		Account: fromAccount,
		Amounts: []structs.TransactionAmount{offerRt, ask},
	})

	toBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, swap.ToAddress.Bytes())
	toAccount := structs.Account{ID: toBech32Addr}

	se.Recipient = append(se.Sender, structs.EventTransfer{
		Account: toAccount,
		Amounts: []structs.TransactionAmount{offerRt, ask},
	})

	se.Amount = map[string]structs.TransactionAmount{
		"offer": offerRt,
		"ask":   ask,
	}

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}
