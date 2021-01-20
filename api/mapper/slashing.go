package mapper

import (
	"errors"
	"fmt"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"
	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/slashing"
)

// SlashingUnjailToSub transforms slashing.MsgUnjail sdk messages to SubsetEvent
func SlashingUnjailToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {
	unjail, ok := msg.(slashing.MsgUnjail)
	if !ok {
		return se, errors.New("Not a unjail type")
	}

	bech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, unjail.ValidatorAddr.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
	}

	return structs.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]structs.Account{"validator": {{ID: bech32ValAddr}}},
	}, nil
}
