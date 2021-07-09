package mapper

import (
	"errors"
	"fmt"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/types"

	"github.com/tendermint/tendermint/libs/bech32"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/distribution"
)

func DistributionWithdrawValidatorCommissionToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	wvc, ok := msg.(distribution.MsgWithdrawValidatorCommission)
	if !ok {
		return se, errors.New("Not a withdraw_validator_commission type")
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, wvc.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"withdraw_validator_commission"},
		Module: "distribution",
		Node:   map[string][]structs.Account{"validator": {{ID: bech32ValAddr}}},
		Recipient: []structs.EventTransfer{{
			Account: structs.Account{ID: bech32ValAddr},
		}},
	}

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}

func DistributionSetWithdrawAddressToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	swa, ok := msg.(distribution.MsgSetWithdrawAddress)
	if !ok {
		return se, errors.New("Not a set_withdraw_address type")
	}

	delegatorBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, swa.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}
	withdrawBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, swa.WithdrawAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting WithdrawAddress: %w", err)
	}

	return structs.SubsetEvent{
		Type:   []string{"set_withdraw_address"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"delegator": {{ID: delegatorBech32ValAddr}},
			"withdraw":  {{ID: withdrawBech32ValAddr}},
		},
	}, nil
}

func DistributionWithdrawDelegatorRewardToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	wdr, ok := msg.(distribution.MsgWithdrawDelegatorReward)
	if !ok {
		return se, errors.New("Not a withdraw_delegator_reward type")
	}

	bech32DelAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, wdr.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, wdr.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"withdraw_delegator_reward"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"delegator": {{ID: bech32DelAddr}},
			"validator": {{ID: bech32ValAddr}},
		},
		Recipient: []structs.EventTransfer{{
			Account: structs.Account{ID: bech32ValAddr},
		}},
	}

	err = produceTransfers(&se, "reward", "", logf)
	return se, err
}

func DistributionFundCommunityPoolToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	fcp, ok := msg.(distributiontypes.MsgFundCommunityPool)
	if !ok {
		return se, errors.New("Not a withdraw_fund_community_pool type")
	}

	bech32DepositerAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, fcp.Depositor.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	evt, err := distributionProduceEvTx(bech32DepositerAddr, fcp.Amount)
	return structs.SubsetEvent{
		Type:   []string{"fund_community_pool"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"depositor": {{ID: bech32DepositerAddr}},
		},
		Sender: []structs.EventTransfer{evt},
	}, err

}

func distributionProduceEvTx(account string, coins sdk.Coins) (evt structs.EventTransfer, err error) {
	evt = structs.EventTransfer{
		Account: structs.Account{ID: account},
	}
	if len(coins) > 0 {
		evt.Amounts = []structs.TransactionAmount{}
		for _, coin := range coins {
			txa := structs.TransactionAmount{
				Currency: coin.Denom,
				Text:     coin.Amount.String(),
			}

			txa.Numeric.Set(coin.Amount.BigInt())
			evt.Amounts = append(evt.Amounts, txa)
		}
	}

	return evt, nil
}
