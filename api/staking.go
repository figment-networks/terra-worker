package api

import (
	"errors"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-project/core/x/staking"
)

const unbondedTokensPoolAddr = "terra1tygms3xhhs3yv487phx3dw4a95jn7t7l8l07dr"

func mapStakingUndelegateToSub(msg sdk.Msg, logf LogFormat) (se structs.SubsetEvent, err error) {
	u, ok := msg.(staking.MsgUndelegate)
	if !ok {
		return se, errors.New("Not a begin_unbonding type")
	}

	se = structs.SubsetEvent{
		Type:   []string{"begin_unbonding"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator": {{ID: u.DelegatorAddress.String()}},
			"validator": {{ID: u.ValidatorAddress.String()}},
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

func mapStakingDelegateToSub(msg sdk.Msg, logf LogFormat) (se structs.SubsetEvent, err error) {
	d, ok := msg.(staking.MsgDelegate)
	if !ok {
		return se, errors.New("Not a delegate type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"delegate"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator": {{ID: d.DelegatorAddress.String()}},
			"validator": {{ID: d.ValidatorAddress.String()}},
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

func mapStakingBeginRedelegateToSub(msg sdk.Msg, logf LogFormat) (se structs.SubsetEvent, err error) {
	br, ok := msg.(staking.MsgBeginRedelegate)
	if !ok {
		return se, errors.New("Not a begin_redelegate type")
	}

	se = structs.SubsetEvent{
		Type:   []string{"begin_redelegate"},
		Module: "staking",
		Node: map[string][]structs.Account{
			"delegator":             {{ID: br.DelegatorAddress.String()}},
			"validator_destination": {{ID: br.ValidatorDstAddress.String()}},
			"validator_source":      {{ID: br.ValidatorDstAddress.String()}},
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

func mapStakingCreateValidatorToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgCreateValidator)
	if !ok {
		return se, errors.New("Not a create_validator type")
	}
	return structs.SubsetEvent{
		Type:   []string{"create_validator"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"delegator": {{ID: ev.DelegatorAddress.String()}},
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
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

func mapStakingEditValidatorToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgEditValidator)
	if !ok {
		return se, errors.New("Not a edit_validator type")
	}
	sev := structs.SubsetEvent{
		Type:   []string{"edit_validator"},
		Module: "distribution",
		Node: map[string][]structs.Account{
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
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
