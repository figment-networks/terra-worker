package mapper

import (
	"errors"
	"strconv"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/terra-project/core/x/evidence"
)

func EvidenceSubmitEvidenceToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {
	mse, ok := msg.(evidence.MsgSubmitEvidence)
	if !ok {
		return se, errors.New("Not a submit_evidence type")
	}

	return structs.SubsetEvent{
		Type:   []string{"submit_evidence"},
		Module: "evidence",
		Node:   map[string][]structs.Account{"submitter": {{ID: mse.Submitter.String()}}},
		Additional: map[string][]string{
			"evidence_consensus":       {mse.Evidence.GetConsensusAddress().String()},
			"evidence_height":          {strconv.FormatInt(mse.Evidence.GetHeight(), 10)},
			"evidence_total_power":     {strconv.FormatInt(mse.Evidence.GetTotalPower(), 10)},
			"evidence_validator_power": {strconv.FormatInt(mse.Evidence.GetValidatorPower(), 10)},
		},
	}, nil
}
