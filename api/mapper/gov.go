package mapper

import (
	"errors"
	"strconv"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/terra-project/core/x/gov"
)

func GovDepositToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	dep, ok := msg.(gov.MsgDeposit)
	if !ok {
		return se, errors.New("Not a deposit type")
	}

	se = structs.SubsetEvent{
		Type:       []string{"deposit"},
		Module:     "gov",
		Node:       map[string][]structs.Account{"depositor": {{ID: dep.Depositor.String()}}},
		Additional: map[string][]string{"proposalID": {strconv.FormatUint(dep.ProposalID, 10)}},
	}

	sender := structs.EventTransfer{Account: structs.Account{ID: dep.Depositor.String()}}
	txAmount := map[string]structs.TransactionAmount{}

	for i, coin := range dep.Amount {
		am := structs.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.Amount.String(),
		}

		sender.Amounts = append(sender.Amounts, am)
		key := "deposit"
		if i > 0 {
			key += "_" + strconv.Itoa(i)
		}

		txAmount[key] = am
	}

	se.Sender = []structs.EventTransfer{sender}
	se.Amount = txAmount

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}

func GovVoteToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	vote, ok := msg.(gov.MsgVote)
	if !ok {
		return se, errors.New("Not a vote type")
	}

	return structs.SubsetEvent{
		Type:   []string{"vote"},
		Module: "gov",
		Node:   map[string][]structs.Account{"voter": {{ID: vote.Voter.String()}}},
		Additional: map[string][]string{
			"proposalID": {strconv.FormatUint(vote.ProposalID, 10)},
			"option":     {vote.Option.String()},
		},
	}, nil
}

func GovSubmitProposalToSub(msg sdk.Msg, logf types.LogFormat) (se structs.SubsetEvent, err error) {
	sp, ok := msg.(gov.MsgSubmitProposal)
	if !ok {
		return se, errors.New("Not a submit_proposal type")
	}

	se = structs.SubsetEvent{
		Type:   []string{"submit_proposal"},
		Module: "gov",
		Node:   map[string][]structs.Account{"proposer": {{ID: sp.Proposer.String()}}},
	}

	sender := structs.EventTransfer{Account: structs.Account{ID: sp.Proposer.String()}}
	txAmount := map[string]structs.TransactionAmount{}

	for i, coin := range sp.InitialDeposit {
		am := structs.TransactionAmount{
			Currency: coin.Denom,
			Numeric:  coin.Amount.BigInt(),
			Text:     coin.Amount.String(),
		}

		sender.Amounts = append(sender.Amounts, am)
		key := "initial_deposit"
		if i > 0 {
			key += "_" + strconv.Itoa(i)
		}

		txAmount[key] = am
	}
	se.Sender = []structs.EventTransfer{sender}
	se.Amount = txAmount

	se.Additional = map[string][]string{}

	if sp.Content.ProposalRoute() != "" {
		se.Additional["proposal_route"] = []string{sp.Content.ProposalRoute()}
	}
	if sp.Content.ProposalType() != "" {
		se.Additional["proposal_type"] = []string{sp.Content.ProposalType()}
	}
	if sp.Content.GetDescription() != "" {
		se.Additional["descritpion"] = []string{sp.Content.GetDescription()}
	}
	if sp.Content.GetTitle() != "" {
		se.Additional["title"] = []string{sp.Content.GetTitle()}
	}
	if sp.Content.String() != "" {
		se.Additional["content"] = []string{sp.Content.String()}
	}

	err = produceTransfers(&se, "send", "", logf)
	return se, err
}
