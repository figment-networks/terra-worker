package mapper

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func BankMultisendToSub(msg []byte, lg types.ABCIMessageLog) (se structs.SubsetEvent, err error) {
	multisend := &bank.MsgMultiSend{}
	if err := proto.Unmarshal(msg, multisend); err != nil {
		return se, fmt.Errorf("Not a multisend type: %w", err)
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

	err = produceTransfers(&se, "send", "", lg)
	return se, err
}

func BankSendToSub(msg []byte, lg types.ABCIMessageLog) (se structs.SubsetEvent, err error) {
	send := &bank.MsgSend{}
	if err := proto.Unmarshal(msg, send); err != nil {
		return se, fmt.Errorf("Not a send type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"send"},
		Module: "bank",
	}

	evt, _ := bankProduceEvTx(send.FromAddress, send.Amount)
	se.Sender = append(se.Sender, evt)

	evt, _ = bankProduceEvTx(send.ToAddress, send.Amount)
	se.Recipient = append(se.Recipient, evt)

	err = produceTransfers(&se, "send", "", lg)
	return se, err
}

func bankProduceEvTx(account string, coins types.Coins) (evt structs.EventTransfer, err error) {
	evt = structs.EventTransfer{
		Account: structs.Account{ID: account},
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

func produceTransfers(se *structs.SubsetEvent, transferType, skipAddr string, lg types.ABCIMessageLog) (err error) {
	var evts []structs.EventTransfer

	for _, ev := range lg.Events {
		if ev.Type != "transfer" {
			continue
		}

		var latestRecipient string
		for _, attr := range ev.GetAttributes() {
			if attr.Key == "recipient" {
				latestRecipient = attr.Value
			}

			if latestRecipient == skipAddr {
				continue
			}

			if attr.Key == "amount" {
				amts := []structs.TransactionAmount{}
				for _, amt := range strings.Split(attr.Value, ",") {
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
					Account: structs.Account{ID: latestRecipient},
				})
			}
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
