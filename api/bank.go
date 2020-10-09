package api

import (
	"errors"

	"github.com/figment-networks/indexer-manager/structs"

	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/bank"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/tendermint/libs/bech32"
)

func mapBankMultisendToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {

	multisend, ok := msg.(bank.MsgMultiSend)
	if !ok {
		return se, errors.New("Not a multisend type")
	}

	se = structs.SubsetEvent{
		Type:   []string{"multisend"},
		Module: "bank",
	}
	for _, i := range multisend.Inputs {
		evt, err := bankProduceEvTx(i.Address, i.Coins)
		if err != nil {
			continue
		}
		se.Sender = append(se.Sender, evt)
	}

	for _, o := range multisend.Outputs {
		evt, err := bankProduceEvTx(o.Address, o.Coins)
		if err != nil {
			continue
		}
		se.Recipient = append(se.Recipient, evt)
	}

	return se, nil
}

func mapBankSendToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {

	send, ok := msg.(bank.MsgSend)
	if !ok {
		return se, errors.New("Not a send type")
	}

	se = structs.SubsetEvent{
		Type:   []string{"send"},
		Module: "bank",
	}

	evt, _ := bankProduceEvTx(send.FromAddress, send.Amount)
	se.Sender = append(se.Sender, evt)

	evt, _ = bankProduceEvTx(send.ToAddress, send.Amount)
	se.Recipient = append(se.Recipient, evt)

	return se, nil
}

func bankProduceEvTx(account sdk.AccAddress, coins sdk.Coins) (evt structs.EventTransfer, err error) {

	bech32Addr := ""
	if !account.Empty() {
		bech32Addr, err = bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, account.Bytes())
	}

	evt = structs.EventTransfer{
		Account: structs.Account{ID: bech32Addr},
	}
	if len(coins) > 0 {
		evt.Amounts = []structs.TransactionAmount{}
		for _, coin := range coins {
			evt.Amounts = append(evt.Amounts, structs.TransactionAmount{
				Currency: coin.Denom,
				Numeric:  coin.Amount.BigInt(),
				Text:     coin.Amount.String(),
			})
		}
	}

	return evt, nil
}
