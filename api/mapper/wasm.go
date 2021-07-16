package mapper

import (
	"fmt"
	"strconv"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/gogo/protobuf/proto"

	wasm "github.com/terra-money/core/x/wasm/types"
)

func WasmExecuteContractToSub(msg []byte) (se structs.SubsetEvent, err error) {
	ec := &wasm.MsgExecuteContract{}
	if err := proto.Unmarshal(msg, ec); err != nil {
		return se, fmt.Errorf("Not a execute_contract type: %w", err)
	}

	evt := structs.EventTransfer{
		Account: structs.Account{ID: ec.Sender},
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

func WasmStoreCodeToSub(msg []byte) (se structs.SubsetEvent, err error) {
	sc := &wasm.MsgStoreCode{}
	if err := proto.Unmarshal(msg, sc); err != nil {
		return se, fmt.Errorf("Not a store_code type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"store_code"},
		Module: "wasm",
		Sender: []structs.EventTransfer{
			{Account: structs.Account{ID: sc.Sender}},
		},
		Additional: map[string][]string{"wasm_byte_code": {string(sc.WASMByteCode)}},
	}

	return se, err
}

func WasmMsgMigrateCodeToSub(msg []byte) (se structs.SubsetEvent, err error) {
	mc := &wasm.MsgMigrateCode{}
	if err := proto.Unmarshal(msg, mc); err != nil {
		return se, fmt.Errorf("Not a migrate_code type: %w", err)
	}

	return structs.SubsetEvent{
		Type:   []string{"migrate_code"},
		Module: "wasm",
		Additional: map[string][]string{
			"wasm_byte_code": {string(mc.WASMByteCode)},
			"code_id":        {strconv.FormatUint(mc.CodeID, 10)},
		},
		Node: map[string][]structs.Account{
			"sender": {{ID: mc.Sender}},
		},
	}, err
}

func WasmMsgUpdateContractAdminToSub(msg []byte) (se structs.SubsetEvent, err error) {
	uco := &wasm.MsgUpdateContractAdmin{}
	if err := proto.Unmarshal(msg, uco); err != nil {
		return se, fmt.Errorf("Not a update_contract_admin type: %w", err)
	}

	return structs.SubsetEvent{
		Type:       []string{"update_contract_admin"},
		Module:     "wasm",
		Additional: map[string][]string{"contract": {uco.Contract}},
		Node: map[string][]structs.Account{
			"new_admin": {{ID: uco.NewAdmin}},
			"admin":     {{ID: uco.Admin}},
		},
	}, err
}

func WasmMsgClearContractAdminToSub(msg []byte) (se structs.SubsetEvent, err error) {
	cca := &wasm.MsgClearContractAdmin{}
	if err := proto.Unmarshal(msg, cca); err != nil {
		return se, fmt.Errorf("Not a clear_contract_admin type: %w", err)
	}

	return structs.SubsetEvent{
		Type:       []string{"clear_contract_admin"},
		Module:     "wasm",
		Additional: map[string][]string{"contract": {cca.Contract}},
		Node: map[string][]structs.Account{
			"admin": {{ID: cca.Admin}},
		},
	}, err
}

func WasmMsgInstantiateContractToSub(msg []byte) (se structs.SubsetEvent, err error) {
	ic := &wasm.MsgInstantiateContract{}
	if err := proto.Unmarshal(msg, ic); err != nil {
		return se, fmt.Errorf("Not a instantiate_contract type: %w", err)
	}

	b, err := ic.InitMsg.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting InitMsg: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"instantiate_contract"},
		Module: "wasm",
		Additional: map[string][]string{
			"code_id":  {strconv.FormatUint(ic.CodeID, 10)},
			"init_msg": {string(b)},
		},
		Node: map[string][]structs.Account{
			"admin":  {{ID: ic.Admin}},
			"sender": {{ID: ic.Sender}},
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

func WasmMsgMigrateContractToSub(msg []byte) (se structs.SubsetEvent, err error) {
	mc := &wasm.MsgMigrateContract{}
	if err := proto.Unmarshal(msg, mc); err != nil {
		return se, fmt.Errorf("Not a migrate_contract type: %w", err)
	}

	b, err := mc.MigrateMsg.MarshalJSON()
	if err != nil {
		return se, fmt.Errorf("error converting ExecuteMsg: %w", err)
	}
	return structs.SubsetEvent{
		Type:   []string{"migrate_contract"},
		Module: "wasm",
		Additional: map[string][]string{
			"contract":    {mc.Contract},
			"new_code_id": {strconv.FormatUint(mc.NewCodeID, 10)},
			"migrate_msg": {string(b)},
		},
		Node: map[string][]structs.Account{
			"admin": {{ID: mc.Admin}},
		},
	}, err

}
