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

		c.logger.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
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

	c.logger.Debug("[COSMOS-API] Sending requests ", zap.Int("number", len(txs)))
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

			// tPath is "/terra.oracle.v1beta1.MsgAggregateExchangeRateVote"
			tPath := strings.Split(m.TypeUrl, ".")
			ev, err := getSubEvent(tPath[1], tPath[3], m, lg)
			if len(ev.Type) > 0 {
				tev.Kind = tPath[3]
				tev.Sub = append(tev.Sub, ev)
			}

			if err != nil {
				if errors.Is(err, errUnknownMessageType) {
					unknownTransactions.WithLabels(m.TypeUrl).Inc()
				} else {
					brokenTransactions.WithLabels(m.TypeUrl).Inc()
				}

				logger.Error("[COSMOS-API] Problem decoding transaction ", zap.Error(err), zap.String("type", tPath[1]), zap.String("route", m.TypeUrl), zap.Int64("height", resp.Height))
				// return trans, err
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

func getSubEvent(msgRoute, msgType string, msg *codec_types.Any, lg types.ABCIMessageLog) (se structs.SubsetEvent, err error) {
	switch msgRoute {
	case "bank":
		switch msgType {
		case "MsgMultiSend":
			return mapper.BankMultisendToSub(msg.Value, lg)
		case "MsgSend":
			return mapper.BankSendToSub(msg.Value, lg)
		}
	case "crisis":
		switch msgType {
		case "MsgVerifyInvariant":
			return mapper.CrisisVerifyInvariantToSub(msg.Value)
		}
	case "distribution":
		switch msgType {
		case "MsgWithdrawValidatorCommission":
			return mapper.DistributionWithdrawValidatorCommissionToSub(msg.Value, lg)
		case "MsgSetWithdrawAddress":
			return mapper.DistributionSetWithdrawAddressToSub(msg.Value)
		case "MsgWithdrawDelegatorReward":
			return mapper.DistributionWithdrawDelegatorRewardToSub(msg.Value, lg)
		case "MsgFundCommunityPool":
			return mapper.DistributionFundCommunityPoolToSub(msg.Value)
		}
	case "evidence":
		switch msgType {
		case "MsgSubmitEvidence":
			return mapper.EvidenceSubmitEvidenceToSub(msg.Value)
		}
	case "gov":
		switch msgType {
		case "MsgDeposit":
			return mapper.GovDepositToSub(msg.Value, lg)
		case "MsgVote":
			return mapper.GovVoteToSub(msg.Value)
		case "MsgSubmitProposal":
			return mapper.GovSubmitProposalToSub(msg.Value, lg)
		}
	case "market":
		switch msgType {
		case "MsgSwap":
			return mapper.MarketSwapToSub(msg.Value, lg)
		case "MsgSwapSend":
			return mapper.MarketSwapSendToSub(msg.Value, lg)
		}
		// deprecated
	// case "msgauth":
	// 	}
	case "oracle": //terra type
		switch msgType {
		// normal prevote and vote are deprecated after columbus-4	https://github.com/terra-money/core/blob/master/x/oracle/spec/04_messages.md
		// case "exchangeratevote":
		// 	return mapper.OracleExchangeRateVoteToSub(msg.Value)
		// case "exchangerateprevote":
		// 	return mapper.OracleExchangeRatePrevoteToSub(msg.Value)
		case "MsgDelegateFeedConsent":
			return mapper.OracleDelegateFeedConsent(msg.Value)
		case "MsgAggregateExchangeRatePrevote":
			return mapper.OracleAggregateExchangeRatePrevoteToSub(msg.Value)
		case "MsgAggregateExchangeRateVote":
			return mapper.OracleAggregateExchangeRateVoteToSub(msg.Value)
		}
	// treasury/vesting? //terra
	case "slashing":
		switch msgType {
		case "MsgUnjail":
			return mapper.SlashingUnjailToSub(msg.Value)
		}
	case "staking":
		switch msgType {
		case "MsgUndelegate":
			return mapper.StakingUndelegateToSub(msg.Value, lg)
		case "MsgEditValidator":
			return mapper.StakingEditValidatorToSub(msg.Value)
		case "MsgCreateValidator":
			return mapper.StakingCreateValidatorToSub(msg.Value)
		case "MsgDelegate":
			return mapper.StakingDelegateToSub(msg.Value, lg)
		case "MsgBeginRedelegate":
			return mapper.StakingBeginRedelegateToSub(msg.Value, lg)
		}
	case "wasm": //terra type
		switch msgType {
		case "MsgExecuteContract":
			return mapper.WasmExecuteContractToSub(msg.Value)
		case "MsgStoreCode":
			return mapper.WasmStoreCodeToSub(msg.Value)
		case "MsgMigrateCode": // new
			return mapper.WasmMsgMigrateCodeToSub(msg.Value)
		case "MsgUpdateContractAdmin": // formerly MsgUpdateContractOwner
			return mapper.WasmMsgUpdateContractAdminToSub(msg.Value) //
		case "MsgClearContractAdmin": // new
			return mapper.WasmMsgClearContractAdminToSub(msg.Value)
		case "MsgInstantiateContract":
			return mapper.WasmMsgInstantiateContractToSub(msg.Value)
		case "MsgMigrateContract":
			return mapper.WasmMsgMigrateContractToSub(msg.Value)
		}
	}

	return se, fmt.Errorf("problem with %s - %s:  %w", msgRoute, msgType, errUnknownMessageType)
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
