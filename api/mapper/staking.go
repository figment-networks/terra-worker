package mapper

import (
	"errors"
	"fmt"

	"github.com/figment-networks/indexer-manager/structs"
	"github.com/figment-networks/terra-worker/api/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"
	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/staking"
)

const unbondedTokensPoolAddr = "terra1tygms3xhhs3yv487phx3dw4a95jn7t7l8l07dr"

// StakingUndelegateToSub transforms staking.MsgUndelegate sdk messages to SubsetEvent
func StakingUndelegateToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	u, ok := msg.(staking.MsgUndelegate)
	if !ok {
		return se, errors.New("Not a begin_unbonding type")
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, u.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	bech32DelAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, u.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"begin_unbonding"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator": {{ID: bech32DelAddr}},
			"validator": {{ID: bech32ValAddr}},
		},
		Amount: map[string]structs.TransactionAmount{
			"undelegate": {
				Currency: u.Amount.Denom,
				Numeric:  u.Amount.Amount.BigInt(),
				Text:     u.Amount.String(),
			},
		},
	}

	err = produceTransfers(&se, "reward", unbondedTokensPoolAddr, logf)
	return se, err
}

// StakingDelegateToSub transforms staking.MsgDelegate sdk messages to SubsetEvent
func StakingDelegateToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	d, ok := msg.(staking.MsgDelegate)
	if !ok {
		return se, errors.New("Not a delegate type")
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, d.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	bech32DelAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, d.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"delegate"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator": {{ID: bech32DelAddr}},
			"validator": {{ID: bech32ValAddr}},
		},
		Amount: map[string]structs.TransactionAmount{
			"delegate": {
				Currency: d.Amount.Denom,
				Numeric:  d.Amount.Amount.BigInt(),
				Text:     d.Amount.String(),
			},
		},
	}

	err = produceTransfers(&se, "reward", "", logf)
	return se, err
}

// StakingBeginRedelegateToSub transforms staking.MsgBeginRedelegate sdk messages to SubsetEvent
func StakingBeginRedelegateToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	br, ok := msg.(staking.MsgBeginRedelegate)
	if !ok {
		return se, errors.New("Not a begin_redelegate type")
	}

	bech32ValDstAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, br.ValidatorDstAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	bech32ValSrcAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, br.ValidatorSrcAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	bech32DelAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, br.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"begin_redelegate"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator":             {{ID: bech32DelAddr}},
			"validator_destination": {{ID: bech32ValDstAddr}},
			"validator_source":      {{ID: bech32ValSrcAddr}},
		},
		Amount: map[string]structs.TransactionAmount{
			"delegate": {
				Currency: br.Amount.Denom,
				Numeric:  br.Amount.Amount.BigInt(),
				Text:     br.Amount.String(),
			},
		},
	}

	err = produceTransfers(&se, "reward", "", logf)
	return se, err
}

// StakingCreateValidatorToSub transforms staking.MsgCreateValidator sdk messages to SubsetEvent
func StakingCreateValidatorToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgCreateValidator)
	if !ok {
		return se, errors.New("Not a create_validator type")
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, ev.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	bech32DelAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, ev.DelegatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting DelegatorAddress: %w", err)
	}

	return structs.SubsetEvent{
		Type:   []string{"create_validator"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"delegator": {{ID: bech32DelAddr}},
			"validator": {
				{
					ID: bech32ValAddr,
					Details: &structs.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
		Amount: map[string]structs.TransactionAmount{
			"self_delegation": {
				Currency: ev.Value.Denom,
				Numeric:  ev.Value.Amount.BigInt(),
				Text:     ev.Value.String(),
			},
			"self_delegation_min": {
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			},
			"commission_rate": {
				Text:    ev.Commission.Rate.String(),
				Numeric: ev.Commission.Rate.Int,
			},
			"commission_max_rate": {
				Text:    ev.Commission.MaxRate.String(),
				Numeric: ev.Commission.MaxRate.Int,
			},
			"commission_max_change_rate": {
				Text:    ev.Commission.MaxChangeRate.String(),
				Numeric: ev.Commission.MaxChangeRate.Int,
			}},
	}, err
}

// StakingEditValidatorToSub transforms staking.MsgEditValidator sdk messages to SubsetEvent
func StakingEditValidatorToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgEditValidator)
	if !ok {
		return se, errors.New("Not a edit_validator type")
	}
	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, ev.ValidatorAddress.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	sev := structs.SubsetEvent{
		Type:   []string{"edit_validator"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"validator": {
				{
					ID: bech32ValAddr,
					Details: &structs.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
	}

	if ev.MinSelfDelegation != nil || ev.CommissionRate != nil {
		sev.Amount = map[string]structs.TransactionAmount{}
		if ev.MinSelfDelegation != nil {
			sev.Amount["self_delegation_min"] = structs.TransactionAmount{
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			}
		}

		if ev.CommissionRate != nil {
			sev.Amount["commission_rate"] = structs.TransactionAmount{
				Text:    ev.CommissionRate.String(),
				Numeric: ev.CommissionRate.Int,
			}
		}
	}
	return sev, err
}
