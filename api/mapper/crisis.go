package mapper

import (
	"fmt"

	shared "github.com/figment-networks/indexing-engine/structs"

	crisis "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/gogo/protobuf/proto"
)

// CrisisVerifyInvariantToSub transforms crisis.MsgVerifyInvariant sdk messages to SubsetEvent
func CrisisVerifyInvariantToSub(msg []byte) (se shared.SubsetEvent, er error) {
	mvi := &crisis.MsgVerifyInvariant{}
	if err := proto.Unmarshal(msg, mvi); err != nil {
		return se, fmt.Errorf("Not a verify_invariant type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"verify_invariant"},
		Module: "crisis",
		Sender: []shared.EventTransfer{{
			Account: shared.Account{ID: mvi.Sender},
		}},
		Additional: map[string][]string{
			"invariant_route":       {mvi.InvariantRoute},
			"invariant_module_name": {mvi.InvariantModuleName},
		},
	}, nil
}
