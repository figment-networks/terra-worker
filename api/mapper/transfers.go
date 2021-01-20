package mapper

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/figment-networks/indexer-manager/structs"
	"github.com/figment-networks/terra-worker/api/types"
	"github.com/figment-networks/terra-worker/api/util"
)

func produceTransfers(se *structs.SubsetEvent, transferType, skipAddr string, logf types.LogFormat) (err error) {
	var evts []structs.EventTransfer
	m := make(map[string][]structs.TransactionAmount)
	for _, ev := range logf.Events {
		if ev.Type != "transfer" {
			continue
		}
		attr := ev.Attributes

		for i, recip := range attr.Recipient {
			if recip == skipAddr || len(attr.Amount) < i {
				continue
			}
			// (pacmessica): split amount because it may contain multiple amounts, eg. from logs `"value": "2896ukrw,16uluna,1umnt"`
			amounts := strings.Split(attr.Amount[i], ",")
			for _, amt := range amounts {
				attrAmt := structs.TransactionAmount{Numeric: &big.Int{}}

				sliced := util.GetCurrency(amt)
				var (
					c       *big.Int
					exp     int32
					coinErr error
				)
				if len(sliced) == 3 {
					attrAmt.Currency = sliced[2]
					c, exp, coinErr = util.GetCoin(sliced[1])
				} else {
					c, exp, coinErr = util.GetCoin(amt)
				}
				if coinErr != nil {
					return fmt.Errorf("[TERRA-API] Error parsing amount '%s': %s ", amt, coinErr)
				}

				attrAmt.Text = amt
				attrAmt.Exp = exp
				attrAmt.Numeric.Set(c)

				m[recip] = append(m[recip], attrAmt)
			}
		}
	}

	for addr, amts := range m {
		evts = append(evts, structs.EventTransfer{
			Amounts: amts,
			Account: structs.Account{ID: addr},
		})
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
