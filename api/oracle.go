package api

import (
	"errors"
	"math/big"

	"github.com/figment-networks/indexer-manager/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/bech32"


	"github.com/terra-project/core/types/util"
	"github.com/terra-project/core/x/oracle"
)

func mapOracleExchangeRateVoteToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {

	exrv, ok := msg.(oracle.MsgExchangeRateVote)
	if !ok {
		return se, errors.New("Not a ExchangeRateVote type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"exchangeratevote"},
		Module: "oracle",
	}

	se.Node = map[string][]structs.Account{}

	if !exrv.Validator.Empty() {
		validatorBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, exrv.Validator.Bytes())
		se.Node["validator"] = []structs.Account{{ID: validatorBech32Addr}}
	}

	if !exrv.Feeder.Empty() {
		feederBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, exrv.Feeder.Bytes())
		se.Node["feeder"] = []structs.Account{{ID: feederBech32Addr}}
	}

	excRt := structs.TransactionAmount{
		Currency: exrv.Denom,
	}
	if exrv.ExchangeRate.Int != nil {
		excRt.Numeric = big.NewInt(0)
		excRt.Numeric.Set(exrv.ExchangeRate.Int)

	}
	se.Amount = map[string]structs.TransactionAmount{"exchangeRate": excRt}

	return se, nil
}

func mapOracleExchangeRatePrevoteToSub(msg sdk.Msg) (se structs.SubsetEvent, er error) {

	exrv, ok := msg.(oracle.MsgExchangeRatePrevote)
	if !ok {
		return se, errors.New("Not a ExchangeRateVote type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"exchangerateprevote"},
		Module: "oracle",
	}

	se.Node = map[string][]structs.Account{}

	if !exrv.Validator.Empty() {
		validatorBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, exrv.Validator.Bytes())
		se.Node["validator"] = []structs.Account{{ID: validatorBech32Addr}}
	}

	if !exrv.Feeder.Empty() {
		feederBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, exrv.Feeder.Bytes())
		se.Node["feeder"] = []structs.Account{{ID: feederBech32Addr}}
	}

	se.Amount = map[string]structs.TransactionAmount{
		"denom": {
			Currency: exrv.Denom,
		},
	}

	return se, nil
}

func mapOracleDelegateFeedConsent(msg sdk.Msg) (se structs.SubsetEvent, er error) {

	dfc, ok := msg.(oracle.MsgDelegateFeedConsent)
	if !ok {
		return se, errors.New("Not a DelegateFeedConsent type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"delegatefeeder"},
		Module: "oracle",
	}

	se.Node = map[string][]structs.Account{}

	if !dfc.Operator.Empty() {
		operatorBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, dfc.Operator.Bytes())
		se.Node["operator"] = []structs.Account{{ID: operatorBech32Addr}}
	}

	if !dfc.Delegate.Empty() {
		feederBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, dfc.Delegate.Bytes())
		se.Node["delegate"] = []structs.Account{{ID: feederBech32Addr}}
	}

	return se, nil
}

func mapOracleAggregateExchangeRatePrevoteToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {

	exrv, ok := msg.(oracle.MsgAggregateExchangeRatePrevote)
	if !ok {
		return se, errors.New("Not a AggregateExchangeRatePrevote type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"aggregateexchangerateprevote"},
		Module: "oracle",
	}

	se.Node = map[string][]structs.Account{}

	if !exrv.Validator.Empty() {
		validatorBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, exrv.Validator.Bytes())
		se.Node["validator"] = []structs.Account{{ID: validatorBech32Addr}}
	}

	if !exrv.Feeder.Empty() {
		feederBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, exrv.Feeder.Bytes())
		se.Node["feeder"] = []structs.Account{{ID: feederBech32Addr}}
	}

	se.Additional = map[string][]string{"hash": {exrv.Hash.String()}}

	return se, nil
}

func mapOracleAggregateExchangeRateVoteToSub(msg sdk.Msg) (se structs.SubsetEvent, err error) {

	exrv, ok := msg.(oracle.MsgAggregateExchangeRateVote)
	if !ok {
		return se, errors.New("Not a AggregateExchangeRatePrevote type")
	}
	se = structs.SubsetEvent{
		Type:   []string{"aggregateexchangerateprevote"},
		Module: "oracle",
	}

	se.Node = map[string][]structs.Account{}

	if !exrv.Validator.Empty() {
		validatorBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixValAddr, exrv.Validator.Bytes())
		se.Node["validator"] = []structs.Account{{ID: validatorBech32Addr}}
	}

	if !exrv.Feeder.Empty() {
		feederBech32Addr, _ := bech32.ConvertAndEncode(util.Bech32PrefixAccAddr, exrv.Feeder.Bytes())
		se.Node["feeder"] = []structs.Account{{ID: feederBech32Addr}}
	}

	se.Additional = map[string][]string{
		"salt":          {exrv.Salt},
		"exchangeRates": {exrv.ExchangeRates},
	}
	return se, nil
}
