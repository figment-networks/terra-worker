package mapper

import (
	"fmt"

	shared "github.com/figment-networks/indexing-engine/structs"

	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/gogo/protobuf/proto"
)

// SlashingUnjailToSub transforms slashing.MsgUnjail sdk messages to SubsetEvent
func SlashingUnjailToSub(msg []byte) (se shared.SubsetEvent, er error) {
	unjail := &slashing.MsgUnjail{}
	if err := proto.Unmarshal(msg, unjail); err != nil {
		return se, fmt.Errorf("Not a unjail type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]shared.Account{"validator": {{ID: unjail.ValidatorAddr}}},
	}, nil
}
