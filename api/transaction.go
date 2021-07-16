package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/mapper"

	codec_types "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	errUnknownMessageType = fmt.Errorf("unknown message type")
)

// TxLogError Error message
type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightHash, block structs.Block, perPage uint64) (txs []structs.Transaction, err error) {
	pag := &query.PageRequest{
		CountTotal: true,
		Limit:      perPage,
	}

	// numberOfItemsInBlock.Add(float64(block.NumberOfTransactions))
	var page = uint64(1)
	for {
		pag.Offset = (perPage * page) - perPage
		now := time.Now()

		if err = c.rateLimiterGRPC.Wait(ctx); err != nil {
			return nil, err
		}

		nctx, cancel := context.WithTimeout(ctx, c.cfg.TimeoutSearchTxCall)
		grpcRes, err := c.txServiceClient.GetTxsEvent(nctx, &tx.GetTxsEventRequest{
			Events:     []string{"tx.height=" + strconv.FormatUint(r.Height, 10)},
			Pagination: pag,
		}, grpc.WaitForReady(true))
		cancel()

		c.logger.Debug("[TERRA-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
		if err != nil {
			// rawRequestGRPCDuration.WithLabels("GetTxsEvent", "error").Observe(time.Since(now).Seconds())
			return nil, err
		}
		// rawRequestGRPCDuration.WithLabels("GetTxsEvent", "ok").Observe(time.Since(now).Seconds())
		// numberOfItemsTransactions.Add(float64(len(grpcRes.Txs)))

		for i, trans := range grpcRes.Txs {
			resp := grpcRes.TxResponses[i]
			// n := time.Now()
			tx, err := rawToTransaction(ctx, c.logger, trans, resp)
			if err != nil {
				return nil, err
			}
			// conversionDuration.WithLabels(resp.Tx.TypeUrl).Observe(time.Since(n).Seconds())
			tx.BlockHash = block.Hash
			tx.ChainID = block.ChainID
			tx.Time = block.Time
			txs = append(txs, tx)
		}

		if grpcRes.Pagination.GetTotal() <= uint64(len(txs)) {
			break
		}

		page++

	}

	c.logger.Debug("[TERRA-API] Sending requests ", zap.Int("number", len(txs)))
	return txs, nil
}

// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, logger *zap.Logger, in *tx.Tx, resp *types.TxResponse) (trans structs.Transaction, err error) {

	trans = structs.Transaction{
		Height:    uint64(resp.Height),
		Hash:      resp.TxHash,
		GasWanted: uint64(resp.GasWanted),
		GasUsed:   uint64(resp.GasUsed),
	}

	if resp.RawLog != "" {
		trans.RawLog = []byte(resp.RawLog)
	} else {
		trans.RawLog = []byte(resp.Logs.String())
	}

	trans.Raw, err = in.Marshal()
	if err != nil {
		return trans, errors.New("Error marshaling tx to raw")
	}

	if in.Body != nil {
		trans.Memo = in.Body.Memo

		for index, m := range in.Body.Messages {
			tev := structs.TransactionEvent{
				ID: strconv.Itoa(index),
			}
			lg := findLog(resp.Logs, index)

			// tPath is "/terra.oracle.v1beta1.MsgAggregateExchangeRateVote" or "/ibc.core.client.v1.MsgCreateClient"
			tPath := strings.Split(m.TypeUrl, ".")
			var err error
			var msgType string

			if len(tPath) == 5 && tPath[0] == "/ibc" {
				msgType = tPath[4]
				err = addIBCSubEvent(tPath[2], msgType, &tev, m, lg)
			} else if len(tPath) == 4 && tPath[0] == "/terra" {
				msgType = tPath[3]
				err = addSubEvent(tPath[1], msgType, &tev, m, lg)
			} else {
				err = fmt.Errorf("TypeURL is in wrong format: %v", m.TypeUrl)
			}

			if err != nil {
				if errors.Is(err, errUnknownMessageType) {
					unknownTransactions.WithLabels(m.TypeUrl).Inc()
				} else {
					brokenTransactions.WithLabels(m.TypeUrl).Inc()
				}

				logger.Error("[TERRA-API] Problem decoding transaction ", zap.Error(err), zap.String("type", tPath[1]), zap.String("route", m.TypeUrl), zap.Int64("height", resp.Height))
				return trans, err
			}

			trans.Events = append(trans.Events, tev)
		}
	}

	if in.AuthInfo != nil {
		for _, coin := range in.AuthInfo.Fee.Amount {
			trans.Fee = append(trans.Fee, structs.TransactionAmount{
				Text:     coin.Amount.String(),
				Numeric:  coin.Amount.BigInt(),
				Currency: coin.Denom,
			})
		}
	}

	if resp.Code > 0 {
		trans.Events = append(trans.Events, structs.TransactionEvent{
			Kind: "error",
			Sub: []structs.SubsetEvent{{
				Type:   []string{"error"},
				Module: resp.Codespace,
				Error: &structs.SubsetEventError{
					Message: resp.RawLog,
				},
			}},
		})
	}

	return trans, nil
}

func addSubEvent(msgRoute, msgType string, tev *structs.TransactionEvent, msg *codec_types.Any, lg types.ABCIMessageLog) (err error) {
	var ev structs.SubsetEvent

	switch msgRoute {
	case "bank":
		switch msgType {
		case "MsgMultiSend":
			ev, err = mapper.BankMultisendToSub(msg.Value, lg)
		case "MsgSend":
			ev, err = mapper.BankSendToSub(msg.Value, lg)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "crisis":
		switch msgType {
		case "MsgVerifyInvariant":
			ev, err = mapper.CrisisVerifyInvariantToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "distribution":
		switch msgType {
		case "MsgWithdrawValidatorCommission":
			ev, err = mapper.DistributionWithdrawValidatorCommissionToSub(msg.Value, lg)
		case "MsgSetWithdrawAddress":
			ev, err = mapper.DistributionSetWithdrawAddressToSub(msg.Value)
		case "MsgWithdrawDelegatorReward":
			ev, err = mapper.DistributionWithdrawDelegatorRewardToSub(msg.Value, lg)
		case "MsgFundCommunityPool":
			ev, err = mapper.DistributionFundCommunityPoolToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "evidence":
		switch msgType {
		case "MsgSubmitEvidence":
			ev, err = mapper.EvidenceSubmitEvidenceToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "gov":
		switch msgType {
		case "MsgDeposit":
			ev, err = mapper.GovDepositToSub(msg.Value, lg)
		case "MsgVote":
			ev, err = mapper.GovVoteToSub(msg.Value)
		case "MsgSubmitProposal":
			ev, err = mapper.GovSubmitProposalToSub(msg.Value, lg)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "market": // terra type
		switch msgType {
		case "MsgSwap":
			ev, err = mapper.MarketSwapToSub(msg.Value, lg)
		case "MsgSwapSend":
			ev, err = mapper.MarketSwapSendToSub(msg.Value, lg)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
		// deprecated
	// case "msgauth":
	// 	}
	case "oracle": //terra type
		switch msgType {
		// normal prevote and vote are deprecated after columbus-4	https://github.com/terra-money/core/blob/master/x/oracle/spec/04_messages.md
		// case "exchangeratevote":
		// 	ev, err = mapper.OracleExchangeRateVoteToSub(msg.Value)
		// case "exchangerateprevote":
		// 	ev, err = mapper.OracleExchangeRatePrevoteToSub(msg.Value)
		case "MsgDelegateFeedConsent":
			ev, err = mapper.OracleDelegateFeedConsent(msg.Value)
		case "MsgAggregateExchangeRatePrevote":
			ev, err = mapper.OracleAggregateExchangeRatePrevoteToSub(msg.Value)
		case "MsgAggregateExchangeRateVote":
			ev, err = mapper.OracleAggregateExchangeRateVoteToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "slashing":
		switch msgType {
		case "MsgUnjail":
			ev, err = mapper.SlashingUnjailToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "staking":
		switch msgType {
		case "MsgUndelegate":
			ev, err = mapper.StakingUndelegateToSub(msg.Value, lg)
		case "MsgEditValidator":
			ev, err = mapper.StakingEditValidatorToSub(msg.Value)
		case "MsgCreateValidator":
			ev, err = mapper.StakingCreateValidatorToSub(msg.Value)
		case "MsgDelegate":
			ev, err = mapper.StakingDelegateToSub(msg.Value, lg)
		case "MsgBeginRedelegate":
			ev, err = mapper.StakingBeginRedelegateToSub(msg.Value, lg)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "wasm": //terra type
		switch msgType {
		case "MsgExecuteContract":
			ev, err = mapper.WasmExecuteContractToSub(msg.Value)
		case "MsgStoreCode":
			ev, err = mapper.WasmStoreCodeToSub(msg.Value)
		case "MsgMigrateCode": // new
			ev, err = mapper.WasmMsgMigrateCodeToSub(msg.Value)
		case "MsgUpdateContractAdmin": // formerly MsgUpdateContractOwner
			ev, err = mapper.WasmMsgUpdateContractAdminToSub(msg.Value) //
		case "MsgClearContractAdmin": // new
			ev, err = mapper.WasmMsgClearContractAdminToSub(msg.Value)
		case "MsgInstantiateContract":
			ev, err = mapper.WasmMsgInstantiateContractToSub(msg.Value)
		case "MsgMigrateContract":
			ev, err = mapper.WasmMsgMigrateContractToSub(msg.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	default:
		err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
	}

	if len(ev.Type) > 0 {
		tev.Sub = append(tev.Sub, ev)
		tev.Kind = ev.Type[0]
	}
	return err
}

func addIBCSubEvent(msgRoute, msgType string, tev *structs.TransactionEvent, m *codec_types.Any, lg types.ABCIMessageLog) (err error) {
	var ev structs.SubsetEvent

	switch msgRoute {
	case "client":
		switch msgType {
		case "MsgCreateClient":
			ev, err = mapper.IBCCreateClientToSub(m.Value)
		case "MsgUpdateClient":
			ev, err = mapper.IBCUpdateClientToSub(m.Value)
		case "MsgUpgradeClient":
			ev, err = mapper.IBCUpgradeClientToSub(m.Value)
		case "MsgSubmitMisbehaviour":
			ev, err = mapper.IBCSubmitMisbehaviourToSub(m.Value)
		default:
			err = fmt.Errorf("problem with %s - %s: %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "connection":
		switch msgType {
		case "MsgConnectionOpenInit":
			ev, err = mapper.IBCConnectionOpenInitToSub(m.Value)
		case "MsgConnectionOpenConfirm":
			ev, err = mapper.IBCConnectionOpenConfirmToSub(m.Value)
		case "MsgConnectionOpenAck":
			ev, err = mapper.IBCConnectionOpenAckToSub(m.Value)
		case "MsgConnectionOpenTry":
			ev, err = mapper.IBCConnectionOpenTryToSub(m.Value)
		default:
			err = fmt.Errorf("problem with %s - %s:  %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "channel":
		switch msgType {
		case "MsgChannelOpenInit":
			ev, err = mapper.IBCChannelOpenInitToSub(m.Value)
		case "MsgChannelOpenTry":
			ev, err = mapper.IBCChannelOpenTryToSub(m.Value)
		case "MsgChannelOpenConfirm":
			ev, err = mapper.IBCChannelOpenConfirmToSub(m.Value)
		case "MsgChannelOpenAck":
			ev, err = mapper.IBCChannelOpenAckToSub(m.Value)
		case "MsgChannelCloseInit":
			ev, err = mapper.IBCChannelCloseInitToSub(m.Value)
		case "MsgChannelCloseConfirm":
			ev, err = mapper.IBCChannelCloseConfirmToSub(m.Value)
		case "MsgRecvPacket":
			ev, err = mapper.IBCChannelRecvPacketToSub(m.Value)
		case "MsgTimeout":
			ev, err = mapper.IBCChannelTimeoutToSub(m.Value)
		case "MsgAcknowledgement":
			ev, err = mapper.IBCChannelAcknowledgementToSub(m.Value)

		default:
			err = fmt.Errorf("problem with %s - %s:  %w", msgRoute, msgType, errUnknownMessageType)
		}
	case "transfer":
		switch msgType {
		case "MsgTransfer":
			ev, err = mapper.IBCTransferToSub(m.Value)
		default:
			err = fmt.Errorf("problem with %s - %s:  %w", msgRoute, msgType, errUnknownMessageType)
		}
	default:
		err = fmt.Errorf("problem with %s - %s:  %w", msgRoute, msgType, errUnknownMessageType)
	}

	if len(ev.Type) > 0 {
		tev.Sub = append(tev.Sub, ev)
		tev.Kind = ev.Type[0]
	}

	return err
}

func findLog(logs types.ABCIMessageLogs, index int) types.ABCIMessageLog {
	if len(logs) <= index {
		return types.ABCIMessageLog{}
	}
	if lg := logs[index]; lg.GetMsgIndex() == uint32(index) {
		return lg
	}
	for _, lg := range logs {
		if lg.GetMsgIndex() == uint32(index) {
			return lg
		}
	}
	return types.ABCIMessageLog{}
}
