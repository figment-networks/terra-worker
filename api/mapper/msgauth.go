package mapper

import (
	"errors"
	"fmt"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"

	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/msgauth"
)

func MsgauthExecAuthorizedToSub(msg sdk.Msg) (se structs.SubsetEvent, msgs []sdk.Msg, err error) {
	execAuthorized, ok := msg.(msgauth.MsgExecAuthorized)
	if !ok {
		return se, nil, errors.New("Not a exec_delegated type")
	}
	bech32Addr := ""
	if !execAuthorized.Grantee.Empty() {
		bech32Addr, err = bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, execAuthorized.Grantee.Bytes())
	}

	return structs.SubsetEvent{
		Type:   []string{"exec_delegated"},
		Module: "msgAuth",
		Node: map[string][]structs.Account{
			"grantee": {{ID: bech32Addr}},
		},
		Sub: []structs.SubsetEvent{},
	}, execAuthorized.Msgs, nil
}

func MsgauthGrantAuthorizationToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	grantAuthorized, ok := msg.(msgauth.MsgGrantAuthorization)
	if !ok {
		return se, errors.New("Not a grant_authorization type")
	}

	subev := structs.SubsetEvent{
		Type:   []string{"grant_authorization"},
		Module: "msgAuth",
		Node:   map[string][]structs.Account{},
		Additional: map[string][]string{
			"type":   {grantAuthorized.Authorization.MsgType()},
			"period": {grantAuthorized.Period.String()},
		},
	}

	grantAuthorized.Period.Seconds()
	if !grantAuthorized.Grantee.Empty() {
		bech32Addr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, grantAuthorized.Grantee.Bytes())
		if err != nil {
			return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
		}
		subev.Node["grantee"] = []structs.Account{{ID: bech32Addr}}
	}
	if !grantAuthorized.Grantee.Empty() {
		bech32Addr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, grantAuthorized.Granter.Bytes())
		if err != nil {
			return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
		}
		subev.Node["granter"] = []structs.Account{{ID: bech32Addr}}
	}

	return subev, nil
}

func MsgauthRevokeAuthorizationToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	revokeAuthorized, ok := msg.(msgauth.MsgRevokeAuthorization)
	if !ok {
		return se, errors.New("Not a revoke_authorization type")
	}
	subev := structs.SubsetEvent{
		Type:   []string{"revoke_authorization"},
		Module: "msgAuth",
		Node:   map[string][]structs.Account{},
		Additional: map[string][]string{
			"type": {revokeAuthorized.AuthorizationMsgType},
		},
	}

	if !revokeAuthorized.Grantee.Empty() {
		bech32Addr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, revokeAuthorized.Grantee.Bytes())
		if err != nil {
			return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
		}
		subev.Node["grantee"] = []structs.Account{{ID: bech32Addr}}
	}
	if !revokeAuthorized.Grantee.Empty() {
		bech32Addr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, revokeAuthorized.Granter.Bytes())
		if err != nil {
			return se, fmt.Errorf("error converting ValidatorAddress: %w", err)
		}
		subev.Node["granter"] = []structs.Account{{ID: bech32Addr}}
	}

	return subev, nil
}
