package mapper

import (
	"fmt"
	"strconv"

	connection "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	"github.com/figment-networks/indexing-engine/structs"
	shared "github.com/figment-networks/indexing-engine/structs"
	"github.com/gogo/protobuf/proto"
)

// IBCConnectionOpenInitToSub transforms ibc.MsgConnectionOpenInit sdk messages to SubsetEvent
func IBCConnectionOpenInitToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenInit{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a connection_open_init type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_init"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"client_id":                  {m.ClientId},
			"version_identifier":         {m.Version.Identifier},
			"version_features":           m.Version.Features,
			"delay_period":               {strconv.FormatUint(m.DelayPeriod, 10)},
			"counterparty_client_id":     {m.Counterparty.ClientId},
			"counterparty_connection_id": {m.Counterparty.ConnectionId},
			"counterparty_prefix":        {string(m.Counterparty.Prefix.String())},
		},
	}, nil
}

// IBCConnectionOpenConfirmToSub transforms ibc.MsgConnectionOpenConfirm sdk messages to SubsetEvent
func IBCConnectionOpenConfirmToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenConfirm{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a connection_open_confirm type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_confirm"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"connection_id":                {m.ConnectionId},
			"proof_ack":                    {string(m.ProofAck)},
			"proof_height_revision_number": {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height": {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCConnectionOpenAckToSub transforms ibc.MsgConnectionOpenAck sdk messages to SubsetEvent
func IBCConnectionOpenAckToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenAck{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a connection_open_ack type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"connection_open_ack"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"connection_id":                    {m.ConnectionId},
			"counterparty_connection_id":       {m.CounterpartyConnectionId},
			"version_identifier":               {m.Version.Identifier},
			"version_features":                 m.Version.Features,
			"client_state":                     {m.ClientState.String()},
			"proof_height_revision_number":     {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":     {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
			"proof_try":                        {string(m.ProofTry)},
			"proof_client":                     {string(m.ProofClient)},
			"proof_consensus":                  {string(m.ProofConsensus)},
			"consensus_height_revision_number": {strconv.FormatUint(m.ConsensusHeight.RevisionNumber, 10)},
			"consensus_height_revision_height": {strconv.FormatUint(m.ConsensusHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCConnectionOpenTryToSub transforms ibc.MsgConnectionOpenTry sdk messages to SubsetEvent
func IBCConnectionOpenTryToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &connection.MsgConnectionOpenTry{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a connection_open_try type: %w", err)
	}

	se = shared.SubsetEvent{
		Type:   []string{"connection_open_try"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"client_id":                        {m.ClientId},
			"previous_connection_id":           {m.PreviousConnectionId},
			"client_state":                     {m.ClientState.String()},
			"counterparty_client_id":           {m.Counterparty.ClientId},
			"counterparty_connection_id":       {m.Counterparty.ConnectionId},
			"counterparty_prefix":              {string(m.Counterparty.Prefix.String())},
			"delay_period":                     {strconv.FormatUint(m.DelayPeriod, 10)},
			"counterparty_versions":            {},
			"proof_height_revision_number":     {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":     {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
			"proof_init":                       {string(m.ProofInit)},
			"proof_client":                     {string(m.ProofClient)},
			"proof_consensus":                  {string(m.ProofConsensus)},
			"consensus_height_revision_number": {strconv.FormatUint(m.ConsensusHeight.RevisionNumber, 10)},
			"consensus_height_revision_height": {strconv.FormatUint(m.ConsensusHeight.RevisionHeight, 10)},
		},
	}

	for i, cpv := range m.CounterpartyVersions {
		se.Additional[fmt.Sprintf("counterparty_version_identifier_%d", i)] = []string{cpv.Identifier}
		se.Additional[fmt.Sprintf("counterparty_version_features_%d", i)] = cpv.Features
	}

	return se, nil
}
