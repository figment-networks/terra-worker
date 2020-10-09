package api

import (
	"errors"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/terra-project/core/x/slashing"
)

func mapSlashingUnjailToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {
	unjail, ok := msg.(slashing.MsgUnjail)
	if !ok {
		return se, errors.New("Not a unjail type")
	}

	return structs.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]structs.Account{"validator": {{ID: unjail.ValidatorAddr.String()}}},
	}, nil
}
