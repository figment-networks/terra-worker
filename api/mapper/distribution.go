package mapper

import (
	"fmt"

	shared "github.com/figment-networks/indexing-engine/structs"

	"github.com/cosmos/cosmos-sdk/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/gogo/protobuf/proto"
)

// DistributionWithdrawValidatorCommissionToSub transforms distribution.MsgWithdrawValidatorCommission sdk messages to SubsetEvent
func DistributionWithdrawValidatorCommissionToSub(msg []byte, lg types.ABCIMessageLog) (se shared.SubsetEvent, err error) {
	wvc := &distribution.MsgWithdrawValidatorCommission{}
	if err := proto.Unmarshal(msg, wvc); err != nil {
		return se, fmt.Errorf("Not a withdraw_validator_commission type: %w", err)
	}

	se = shared.SubsetEvent{
		Type:   []string{"withdraw_validator_commission"},
		Module: "distribution",
		Node:   map[string][]shared.Account{"validator": {{ID: wvc.ValidatorAddress}}},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wvc.ValidatorAddress},
		}},
	}

	err = produceTransfers(&se, "send", "", lg)
	return se, err
}

// DistributionSetWithdrawAddressToSub transforms distribution.MsgSetWithdrawAddress sdk messages to SubsetEvent
func DistributionSetWithdrawAddressToSub(msg []byte) (se shared.SubsetEvent, err error) {
	swa := &distribution.MsgSetWithdrawAddress{}
	if err := proto.Unmarshal(msg, swa); err != nil {
		return se, fmt.Errorf("Not a set_withdraw_address type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"set_withdraw_address"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: swa.DelegatorAddress}},
			"withdraw":  {{ID: swa.WithdrawAddress}},
		},
	}, nil
}

// DistributionWithdrawDelegatorRewardToSub transforms distribution.MsgWithdrawDelegatorReward sdk messages to SubsetEvent
func DistributionWithdrawDelegatorRewardToSub(msg []byte, lg types.ABCIMessageLog) (se shared.SubsetEvent, err error) {
	wdr := &distribution.MsgWithdrawDelegatorReward{}
	if err := proto.Unmarshal(msg, wdr); err != nil {
		return se, fmt.Errorf("Not a withdraw_delegator_reward type: %w", err)
	}

	se = shared.SubsetEvent{
		Type:   []string{"withdraw_delegator_reward"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: wdr.DelegatorAddress}},
			"validator": {{ID: wdr.ValidatorAddress}},
		},
		Recipient: []shared.EventTransfer{{
			Account: shared.Account{ID: wdr.ValidatorAddress},
		}},
	}

	err = produceTransfers(&se, "reward", "", lg)
	return se, err
}

// DistributionFundCommunityPoolToSub transforms distribution.MsgFundCommunityPool sdk messages to SubsetEvent
func DistributionFundCommunityPoolToSub(msg []byte) (se shared.SubsetEvent, err error) {
	fcp := &distribution.MsgFundCommunityPool{}
	if err := proto.Unmarshal(msg, fcp); err != nil {
		return se, fmt.Errorf("Not a fund_community_pool type: %w", err)
	}

	evt, err := distributionProduceEvTx(fcp.Depositor, fcp.Amount)
	return shared.SubsetEvent{
		Type:   []string{"fund_community_pool"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"depositor": {{ID: fcp.Depositor}},
		},
		Sender: []shared.EventTransfer{evt},
	}, err

}

func distributionProduceEvTx(account string, coins types.Coins) (evt shared.EventTransfer, err error) {

	evt = shared.EventTransfer{
		Account: shared.Account{ID: account},
	}
	if len(coins) > 0 {
		evt.Amounts = []shared.TransactionAmount{}
		for _, coin := range coins {
			txa := shared.TransactionAmount{
				Currency: coin.Denom,
				Text:     coin.Amount.String(),
			}

			txa.Numeric.Set(coin.Amount.BigInt())
			evt.Amounts = append(evt.Amounts, txa)
		}
	}

	return evt, nil
}
