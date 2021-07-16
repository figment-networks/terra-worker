package mapper

import (
	"fmt"
	"strings"

	"github.com/figment-networks/indexing-engine/structs"

	"github.com/gogo/protobuf/proto"
	oracle "github.com/terra-money/core/x/oracle/types"
)

func OracleDelegateFeedConsent(msg []byte) (se structs.SubsetEvent, er error) {
	dfc := &oracle.MsgDelegateFeedConsent{}
	if err := proto.Unmarshal(msg, dfc); err != nil {
		return se, fmt.Errorf("Not a delegatefeeder type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"delegatefeeder"},
		Module: "oracle",
		Node: map[string][]structs.Account{
			"operator": {{ID: dfc.Operator}},
			"delegate": {{ID: dfc.Delegate}},
		},
	}

	return se, nil
}

func OracleAggregateExchangeRatePrevoteToSub(msg []byte) (se structs.SubsetEvent, err error) {
	exrv := &oracle.MsgAggregateExchangeRatePrevote{}
	if err := proto.Unmarshal(msg, exrv); err != nil {
		return se, fmt.Errorf("Not a aggregateexchangerateprevote type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"aggregateexchangerateprevote"},
		Module: "oracle",
		Node: map[string][]structs.Account{
			"validator": {{ID: exrv.Validator}},
			"feeder":    {{ID: exrv.Feeder}},
		},
		Additional: map[string][]string{"hash": {exrv.Hash}},
	}

	return se, nil
}

func OracleAggregateExchangeRateVoteToSub(msg []byte) (se structs.SubsetEvent, err error) {
	exrv := &oracle.MsgAggregateExchangeRateVote{}
	if err := proto.Unmarshal(msg, exrv); err != nil {
		return se, fmt.Errorf("Not a aggregateexchangeratevote type: %w", err)
	}

	se = structs.SubsetEvent{
		Type:   []string{"aggregateexchangeratevote"},
		Module: "oracle",
		Node: map[string][]structs.Account{
			"validator": {{ID: exrv.Validator}},
			"feeder":    {{ID: exrv.Feeder}},
		},
		Additional: map[string][]string{
			"salt":          {exrv.Salt},
			"exchangeRates": strings.Split(exrv.ExchangeRates, ","),
		},
	}

	return se, nil
}
