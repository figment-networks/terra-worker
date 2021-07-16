package mapper

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/bank"

	"github.com/tendermint/tendermint/libs/bech32"
)

func BankMultisendToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
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

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}

func BankSendToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
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

	err = produceTransfers(&se, "send", "", logf)
	return se, err
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

func produceTransfers(se *structs.SubsetEvent, transferType, skipAddr string, logf types.LogFormat) (err error) {
	var evts []structs.EventTransfer

	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}
		attr := ev.Attributes

		for i, recip := range attr.Recipient {
			if recip == skipAddr || len(attr.Amount) < i {
				continue
			}
			amts := []structs.TransactionAmount{}

			for _, amt := range strings.Split(attr.Amount[i], ",") { // (pacmessica): split amount because it may contain multiple amounts, eg. from logs `"value": "2896ukrw,16uluna,1umnt"`
				attrAmt := structs.TransactionAmount{Numeric: &big.Int{}}

				sliced := getCurrency(amt)
				var (
					c       *big.Int
					exp     int32
					coinErr error
				)
				if len(sliced) == 3 {
					attrAmt.Currency = sliced[2]
					c, exp, coinErr = getCoin(sliced[1])
				} else {
					c, exp, coinErr = getCoin(amt)
				}
				if coinErr != nil {
					return fmt.Errorf("[TERRA-API] Error parsing amount '%s': %s ", amt, coinErr)
				}

				attrAmt.Text = amt
				attrAmt.Exp = exp
				attrAmt.Numeric.Set(c)

				amts = append(amts, attrAmt)
			}
			evts = append(evts, structs.EventTransfer{
				Amounts: amts,
				Account: structs.Account{ID: recip},
			})
		}
	}

	if len(evts) <= 0 {
		return
	}

	if se.Transfers[transferType] == nil {
		se.Transfers = make(map[string][]structs.EventTransfer)
	}
	se.Transfers[transferType] = evts

	return
}

func getCoin(s string) (number *big.Int, exp int32, err error) {
	s = strings.Replace(s, ",", ".", -1)
	strs := strings.Split(s, `.`)
	number = &big.Int{}
	if len(strs) == 1 {
		number.SetString(strs[0], 10)
		return number, 0, nil
	}
	if len(strs) == 2 {
		number.SetString(strs[0]+strs[1], 10)
		return number, int32(len(strs[1])), nil
	}

	return number, 0, errors.New("Impossible to parse ")
}

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-]+)[\\s]*([^0-9\\s]+)$")

func getCurrency(in string) []string {
	return curencyRegex.FindStringSubmatch(in)
}
