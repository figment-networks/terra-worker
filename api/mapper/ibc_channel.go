package mapper

import (
	"fmt"
	"strconv"

	"github.com/figment-networks/indexing-engine/structs"
	shared "github.com/figment-networks/indexing-engine/structs"

	channel "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/gogo/protobuf/proto"
)

// IBCChannelOpenInitToSub transforms ibc.MsgChannelOpenInit sdk messages to SubsetEvent
func IBCChannelOpenInitToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelOpenInit{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_open_init type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_open_init"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":                         {m.PortId},
			"channel_state":                   {strconv.FormatInt(int64(m.Channel.State), 10)},
			"channel_ordering":                {strconv.FormatInt(int64(m.Channel.Ordering), 10)},
			"channel_counterparty_port_id":    {m.Channel.Counterparty.PortId},
			"channel_counterparty_channel_id": {m.Channel.Counterparty.ChannelId},
			"channel_connection_hops":         m.Channel.ConnectionHops,
			"channel_version":                 {m.Channel.Version},
		},
	}, nil
}

// IBCChannelOpenConfirmToSub transforms ibc.MsgChannelOpenConfirm sdk messages to SubsetEvent
func IBCChannelOpenConfirmToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelOpenConfirm{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_open_confirm type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_open_confirm"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":                      {m.PortId},
			"channel_id":                   {m.ChannelId},
			"proof_ack":                    {string(m.ProofAck)},
			"proof_height_revision_number": {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height": {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCChannelOpenAckToSub transforms ibc.MsgChannelOpenAck sdk messages to SubsetEvent
func IBCChannelOpenAckToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelOpenAck{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_open_ack type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_open_ack"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":                      {m.PortId},
			"channel_id":                   {m.ChannelId},
			"counterparty_channel_id":      {m.CounterpartyChannelId},
			"counterparty_version":         {m.CounterpartyVersion},
			"proof_try":                    {string(m.ProofTry)},
			"proof_height_revision_number": {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height": {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCChannelOpenTryToSub transforms ibc.MsgChannelOpenTry sdk messages to SubsetEvent
func IBCChannelOpenTryToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelOpenTry{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_open_try type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_open_try"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":                         {m.PortId},
			"previous_channel_id":             {m.PreviousChannelId},
			"channel_state":                   {strconv.FormatInt(int64(m.Channel.State), 10)},
			"channel_ordering":                {strconv.FormatInt(int64(m.Channel.Ordering), 10)},
			"channel_counterparty_port_id":    {m.Channel.Counterparty.PortId},
			"channel_counterparty_channel_id": {m.Channel.Counterparty.ChannelId},
			"channel_connection_hops":         m.Channel.ConnectionHops,
			"channel_version":                 {m.Channel.Version},
			"counterparty_version":            {m.CounterpartyVersion},
			"proof_init":                      {string(m.ProofInit)},
			"proof_height_revision_number":    {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":    {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCChannelCloseInitToSub transforms ibc.MsgChannelCloseInit sdk messages to SubsetEvent
func IBCChannelCloseInitToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelCloseInit{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_close_init type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_close_init"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":    {m.PortId},
			"channel_id": {m.ChannelId},
		},
	}, nil
}

// IBCChannelCloseConfirmToSub transforms ibc.MsgChannelCloseConfirm sdk messages to SubsetEvent
func IBCChannelCloseConfirmToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgChannelCloseConfirm{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_close_confirm type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_close_confirm"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"port_id":                      {m.PortId},
			"channel_id":                   {m.ChannelId},
			"proof_init":                   {string(m.ProofInit)},
			"proof_height_revision_number": {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height": {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCChannelRecvPacketToSub transforms ibc.MsgRecvPacket sdk messages to SubsetEvent
func IBCChannelRecvPacketToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgRecvPacket{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a recv_packet type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"recv_packet"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"packet_sequence":                       {strconv.FormatUint(m.Packet.Sequence, 10)},
			"packet_source_port":                    {m.Packet.SourcePort},
			"packet_source_channel":                 {m.Packet.SourceChannel},
			"packet_destination_port":               {m.Packet.DestinationPort},
			"packet_destination_channel":            {m.Packet.DestinationChannel},
			"packet_data":                           {string(m.Packet.Data)},
			"packet_timeout_height_revision_number": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionNumber, 10)},
			"packet_timeout_height_revision_height": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionHeight, 10)},
			"packet_timeout_stamp":                  {strconv.FormatUint(m.Packet.TimeoutTimestamp, 10)},
			"proof_commitment":                      {string(m.ProofCommitment)},
			"proof_height_revision_number":          {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":          {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}

// IBCChannelTimeoutToSub transforms ibc.MsgTimeout sdk messages to SubsetEvent
func IBCChannelTimeoutToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgTimeout{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a timeout type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"timeout"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"packet_sequence":                       {strconv.FormatUint(m.Packet.Sequence, 10)},
			"packet_source_port":                    {m.Packet.SourcePort},
			"packet_source_channel":                 {m.Packet.SourceChannel},
			"packet_destination_port":               {m.Packet.DestinationPort},
			"packet_destination_channel":            {m.Packet.DestinationChannel},
			"packet_data":                           {string(m.Packet.Data)},
			"packet_timeout_height_revision_number": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionNumber, 10)},
			"packet_timeout_height_revision_height": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionHeight, 10)},
			"packet_timeout_stamp":                  {strconv.FormatUint(m.Packet.TimeoutTimestamp, 10)},
			"proof_unreceived":                      {string(m.ProofUnreceived)},
			"proof_height_revision_number":          {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":          {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
			"next_sequence_recv":                    {strconv.FormatUint(m.NextSequenceRecv, 10)},
		},
	}, nil
}

// IBCChannelAcknowledgementToSub transforms ibc.MsgAcknowledgement sdk messages to SubsetEvent
func IBCChannelAcknowledgementToSub(msg []byte) (se shared.SubsetEvent, err error) {
	m := &channel.MsgAcknowledgement{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return se, fmt.Errorf("Not a channel_acknowledgement type: %w", err)
	}

	return shared.SubsetEvent{
		Type:   []string{"channel_acknowledgement"},
		Module: "ibc",
		Node: map[string][]structs.Account{
			"signer": {{ID: m.Signer}},
		},
		Additional: map[string][]string{
			"packet_sequence":                       {strconv.FormatUint(m.Packet.Sequence, 10)},
			"packet_source_port":                    {m.Packet.SourcePort},
			"packet_source_channel":                 {m.Packet.SourceChannel},
			"packet_destination_port":               {m.Packet.DestinationPort},
			"packet_destination_channel":            {m.Packet.DestinationChannel},
			"packet_data":                           {string(m.Packet.Data)},
			"packet_timeout_height_revision_number": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionNumber, 10)},
			"packet_timeout_height_revision_height": {strconv.FormatUint(m.Packet.TimeoutHeight.RevisionHeight, 10)},
			"packet_timeout_stamp":                  {strconv.FormatUint(m.Packet.TimeoutTimestamp, 10)},
			"acknowledgement":                       {string(m.Acknowledgement)},
			"proof_acked":                           {string(m.ProofAcked)},
			"proof_height_revision_number":          {strconv.FormatUint(m.ProofHeight.RevisionNumber, 10)},
			"proof_height_revision_height":          {strconv.FormatUint(m.ProofHeight.RevisionHeight, 10)},
		},
	}, nil
}
