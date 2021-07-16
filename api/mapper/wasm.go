package mapper

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/figment-networks/indexing-engine/structs"

	"github.com/tendermint/tendermint/libs/bech32"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/wasm"
)

func WasmExecuteContractToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ec, ok := msg.(wasm.MsgExecuteContract)
	if !ok {
		return se, errors.New("Not a execute_contract type")
	}

	senderBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, ec.Sender.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting Sender address: %w", err)
	}

	evt := structs.EventTransfer{
		Account: structs.Account{ID: senderBech32ValAddr},
	}
	if len(ec.Coins) > 0 {
		evt.Amounts = []structs.TransactionAmount{}
		for _, coin := range ec.Coins {
			evt.Amounts = append(evt.Amounts, structs.TransactionAmount{
				Currency: coin.Denom,
				Numeric:  coin.Amount.BigInt(),
				Text:     coin.Amount.String(),
			})
		}
	}

	b, err := ec.ExecuteMsg.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting ExecuteMsg: %w", err)
	}

	return structs.SubsetEvent{
		Type:       []string{"execute_contract"},
		Module:     "wasm",
		Sender:     []structs.EventTransfer{evt},
		Additional: map[string][]string{"execute_message": {string(b)}},
	}, err
}

func WasmStoreCodeToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	sc, ok := msg.(wasm.MsgStoreCode)
	if !ok {
		return se, errors.New("Not a store_code type")
	}

	senderBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, sc.Sender.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting Sender address: %w", err)
	}

	b, err := sc.WASMByteCode.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting ExecuteMsg: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"store_code"},
		Module: "wasm",
		Sender: []structs.EventTransfer{
			{Account: structs.Account{ID: senderBech32ValAddr}},
		},
		Additional: map[string][]string{"wasm_byte_code": {string(b)}},
	}

	return se, err
}

func WasmMsgUpdateContractOwnerToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	uco, ok := msg.(wasm.MsgUpdateContractOwner)
	if !ok {
		return se, errors.New("Not a update_contract_owner type")
	}

	contractBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, uco.Contract.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting cntract address: %w", err)
	}

	newOwnerBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, uco.NewOwner.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting new owner address: %w", err)
	}

	ownerBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, uco.Owner.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting owner address: %w", err)
	}

	return structs.SubsetEvent{
		Type:       []string{"update_contract_owner"},
		Module:     "wasm",
		Additional: map[string][]string{"contract": {contractBech32ValAddr}},
		Node: map[string][]structs.Account{
			"new_owner": {{ID: newOwnerBech32ValAddr}},
			"owner":     {{ID: ownerBech32ValAddr}},
		},
	}, err
}

func WasmMsgInstantiateContractToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	ic, ok := msg.(wasm.MsgInstantiateContract)
	if !ok {
		return se, errors.New("Not a instantiate_contract type")
	}

	ownerBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, ic.Owner.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting owner address: %w", err)
	}

	b, err := ic.InitMsg.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting InitMsg: %w", err)
	}

	migratable := "false"
	if ic.Migratable {
		migratable = "true"
	}

	se = structs.SubsetEvent{
		Type:   []string{"instantiate_contract"},
		Module: "wasm",
		Additional: map[string][]string{
			"migratable": {migratable},
			"code_id":    {strconv.FormatUint(ic.CodeID, 10)},
			"init_msg":   {string(b)},
		},
		Node: map[string][]structs.Account{
			"owner": {{ID: ownerBech32ValAddr}},
		},
		Amount: map[string]structs.TransactionAmount{},
	}

	if len(ic.InitCoins) > 0 {
		for i, coin := range ic.InitCoins {
			se.Amount["init_coin_"+strconv.Itoa(i)] = structs.TransactionAmount{
				Currency: coin.Denom,
				Numeric:  coin.Amount.BigInt(),
				Text:     coin.Amount.String(),
			}
		}
	}

	return se, err
}

func WasmMsgMigrateContractToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	mc, ok := msg.(wasm.MsgMigrateContract)
	if !ok {
		return se, errors.New("Not a migrate_contract type")
	}

	contractBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, mc.Contract.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting contract address: %w", err)
	}

	ownerBech32ValAddr, err := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, mc.Owner.Bytes())
	if err != nil {
		return se, fmt.Errorf("error converting owner address: %w", err)
	}

	b, err := mc.MigrateMsg.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting ExecuteMsg: %w", err)
	}
	return structs.SubsetEvent{
		Type:   []string{"migrate_contract"},
		Module: "wasm",
		Additional: map[string][]string{
			"contract":    {contractBech32ValAddr},
			"new_code_id": {strconv.FormatUint(mc.NewCodeID, 10)},
			"migrate_msg": {string(b)},
		},
		Node: map[string][]structs.Account{
			"owner": {{ID: ownerBech32ValAddr}},
		},
	}, err

}
