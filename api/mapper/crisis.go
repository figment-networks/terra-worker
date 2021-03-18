package mapper

import (
	"errors"
	"fmt"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"

	"github.com/terra-project/core/types/util"
	crisis "github.com/terra-project/core/x/crisis"
)

func CrisisVerifyInvariantToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {
	mvi, ok := msg.(crisis.MsgVerifyInvariant)
	if !ok {
		return se, errors.New("Not a verify_invariant type")
	}

	bech32Addr := ""
	if !mvi.Sender.Empty() {
		var err error
		bech32Addr, err = bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, mvi.Sender.Bytes())
		if err != nil {
			return se, fmt.Errorf("error converting Sender: %w", err)
		}
	}

	return structs.SubsetEvent{
		Type:   []string{"verify_invariant"},
		Module: "crisis",
		Sender: []structs.EventTransfer{{
			Account: structs.Account{ID: bech32Addr},
		}},
		Additional: map[string][]string{
			"invariant_route":       {mvi.InvariantRoute},
			"invariant_module_name": {mvi.InvariantModuleName},
		},
	}, nil
}
