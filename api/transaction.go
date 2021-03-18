package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/figment-networks/indexer-manager/structs"
	cStruct "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/figment-networks/terra-worker/api/mapper"
	"github.com/figment-networks/terra-worker/api/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	amino "github.com/tendermint/go-amino"
	"github.com/terra-project/core/x/auth"

	"go.uber.org/zap"
)

// TxLogError Error message
type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightRange, chain_id string, blocks map[uint64]structs.Block, out chan cStruct.OutResp, page, perPage int, fin chan string) {
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		fin <- err.Error()
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()

	s := strings.Builder{}

	s.WriteString(`"`)
	s.WriteString("tx.height>=")
	s.WriteString(strconv.FormatUint(r.StartHeight, 10))

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString(" AND ")
		s.WriteString("tx.height<=")
		s.WriteString(strconv.FormatUint(r.EndHeight, 10))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	// (lukanus): do not block initial calls
	if r.EndHeight != 0 && r.StartHeight != 0 {
		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			fin <- err.Error()
			return
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	c.logger.Debug("[TERRA-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		fin <- err.Error()
		return
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[TERRA-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		fin <- err.Error()
		return
	}

	rawRequestHTTPDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &types.ResultTxSearch{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[TERRA-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		fin <- err.Error()
		return
	}

	if result.Error.Message != "" {
		c.logger.Error("[TERRA-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		fin <- err.Error()
		return
	}

	if result.TotalCount != "" {
		if err != nil {
			c.logger.Error("[TERRA-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
			fin <- err.Error()
			return
		}
	}

	c.logger.Debug("[TERRA-API] Converting requests ", zap.Int("number", len(result.Txs)), zap.Int("blocks", len(blocks)))
	err = RawToTransaction(c.logger, c.cdc, result.Txs, blocks, out)
	if err != nil {
		c.logger.Error("[TERRA-API] Error getting rawToTransaction", zap.Error(err))
		fin <- err.Error()
	}
	c.logger.Debug("[TERRA-API] Converted all requests ")

	fin <- ""

	return
}

func RawToTransaction(logger *zap.Logger, cdc *amino.Codec, in []types.TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp) error {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for _, txRaw := range in {
		readr.Reset(txRaw.TxResult.Log)
		lf := []types.LogFormat{}
		txErr := TxLogError{}
		if err := dec.Decode(&lf); err != nil {
			// (lukanus): Try to fallback to known error format
			dec = json.NewDecoder(readr) // (lukanus): reassign decoder in case of failure
			txErr.Message = txRaw.TxResult.Log
		}

		tx, err := rawToTransaction(logger, cdc, txRaw, lf, txErr, blocks)
		if err != nil {
			return err
		}
		out <- tx
	}

	return nil
}

func RawToTransactionCh(logger *zap.Logger, cdc *amino.Codec, wg *sync.WaitGroup, in <-chan types.TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp) {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	defer wg.Done()
	var err error
	for txRaw := range in {
		lf := []types.LogFormat{}
		txErr := TxLogError{}
		if txRaw.TxResult.Log != "" {
			readr.Reset(txRaw.TxResult.Log)
			if err = dec.Decode(&lf); err != nil {
				dec = json.NewDecoder(readr) // (lukanus): reassign decoder in case of failure

				// (lukanus): Try to fallback to known error format
				txErr.Message = txRaw.TxResult.Log
			}
		}
		tx, err := rawToTransaction(logger, cdc, txRaw, lf, txErr, blocks)
		if err != nil {
			logger.Error("[TERRA-API] Problem decoding raw transaction", zap.Error(err), zap.String("height", txRaw.Height), zap.String("hash", txRaw.Hash))
		}
		out <- tx
	}
}

func rawToTransaction(logger *zap.Logger, cdc *amino.Codec, txRaw types.TxResponse, txLog []types.LogFormat, txErr TxLogError, blocks map[uint64]structs.Block) (cStruct.OutResp, error) {
	timer := metrics.NewTimer(transactionConversionDuration)
	defer timer.ObserveDuration()

	numberOfItemsTransactions.Inc()

	tx := &auth.StdTx{}
	txReader := strings.NewReader(txRaw.TxData)
	base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)

	_, err := cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
	if err != nil {
		txReader := strings.NewReader(txRaw.TxData)
		base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)
		logger.Error("[TERRA-API] Problem decoding raw transaction (cdc) ", zap.Error(err), zap.String("height", txRaw.Height))
		_, err := cdcA.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
		if err != nil {

		}
	}
	hInt, err := strconv.ParseUint(txRaw.Height, 10, 64)
	if err != nil {
		logger.Error("[TERRA-API] Problem parsing height", zap.Error(err), zap.String("height", txRaw.Height))
	}

	outTX := cStruct.OutResp{Type: "Transaction"}
	block := blocks[hInt]

	trans := structs.Transaction{
		Hash:      txRaw.Hash,
		Memo:      tx.GetMemo(),
		Time:      block.Time,
		BlockHash: block.Hash,
		ChainID:   block.ChainID,
		Height:    hInt,
	}
	trans.GasWanted, err = strconv.ParseUint(txRaw.TxResult.GasWanted, 10, 64)
	if err != nil {
		outTX.Error = err
	}
	trans.GasUsed, err = strconv.ParseUint(txRaw.TxResult.GasUsed, 10, 64)
	if err != nil {
		outTX.Error = err
	}

	txReader.Seek(0, 0)
	trans.Raw = make([]byte, txReader.Len())
	txReader.Read(trans.Raw)

	txLogReader := strings.NewReader(txRaw.TxResult.Log)
	trans.RawLog = make([]byte, txLogReader.Len())
	txLogReader.Read(trans.RawLog)

	for _, coin := range tx.Fee.Amount {
		trans.Fee = append(trans.Fee, structs.TransactionAmount{
			Text:     coin.Amount.String(),
			Numeric:  coin.Amount.BigInt(),
			Currency: coin.Denom,
		})
	}

	presentIndexes := map[string]bool{}

	for index, msg := range tx.Msgs {
		tev := structs.TransactionEvent{
			ID: strconv.Itoa(index),
		}
		lf := findLog(txLog, index)
		ev, err := getSubEvent(msg, lf)
		if len(ev.Type) > 0 {
			tev.Kind = msg.Type()
			tev.Sub = append(tev.Sub, ev)
		}

		if err != nil {
			logger.Error("[TERRA-API] Problem decoding transaction ", zap.Error(err), zap.Uint64("height", trans.Height), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
			continue
		}

		trans.Events = append(trans.Events, tev)
		// (lukanus): set this only for successfull
		presentIndexes[tev.ID] = true
	}

	for _, logf := range txLog {
		msgIndex := strconv.FormatFloat(logf.MsgIndex, 'f', -1, 64)
		if _, ok := presentIndexes[msgIndex]; ok {
			continue
		}

		tev := eventFromLogs(logf)

		// (lukanus): if call was an error append error message from the log
		if !logf.Success {
			subsError := &structs.SubsetEventError{
				Message: logf.Log.Message,
			}

			if len(tev.Sub) > 0 {
				if tev.Sub[0].Error == nil { // do not overwrite
					tev.Sub[0].Error = subsError
				}
			} else {
				tev.Sub = append(tev.Sub, structs.SubsetEvent{Error: subsError})
			}

		}

		trans.Events = append(trans.Events, tev)
	}

	if txErr.Message != "" {
		tev := structs.TransactionEvent{
			Kind: "error",
			Sub: []structs.SubsetEvent{{
				Type:   []string{"error"},
				Module: txErr.Codespace,
				Error:  &structs.SubsetEventError{Message: txErr.Message},
			}},
		}
		trans.Events = append(trans.Events, tev)
	}
	outTX.Payload = trans

	return outTX, nil
}

func eventFromLogs(lf types.LogFormat) structs.TransactionEvent {

	te := structs.TransactionEvent{
		ID: strconv.FormatFloat(lf.MsgIndex, 'f', -1, 64),
	}

	for _, ev := range lf.Events {
		if ev.Attributes != nil {
			sub := structs.SubsetEvent{
				Type:   []string{ev.Attributes.Action},
				Module: ev.Attributes.Module,
			}
			if len(ev.Attributes.Sender) > 0 {
				for _, s := range ev.Attributes.Sender {
					sub.Sender = append(sub.Sender, structs.EventTransfer{
						Account: structs.Account{ID: s},
					})
				}
			}

			if len(ev.Attributes.Recipient) > 0 {
				for _, r := range ev.Attributes.Recipient {
					sub.Recipient = append(sub.Recipient, structs.EventTransfer{
						Account: structs.Account{ID: r},
					})
				}
			}
			te.Sub = append(te.Sub, sub)
		}
	}

	return te
}

func getSubEvent(msg sdk.Msg, lf types.LogFormat) (se structs.SubsetEvent, err error) {
	switch msg.Route() {
	case "bank":
		switch msg.Type() {
		case "multisend":
			return mapper.BankMultisendToSub(msg, lf)
		case "send":
			return mapper.BankSendToSub(msg, lf)
		}
	case "crisis":
		switch msg.Type() {
		case "verify_invariant":
			return mapper.CrisisVerifyInvariantToSub(msg)
		}
	case "distribution":
		switch msg.Type() {
		case "withdraw_validator_commission":
			return mapper.DistributionWithdrawValidatorCommissionToSub(msg, lf)
		case "set_withdraw_address":
			return mapper.DistributionSetWithdrawAddressToSub(msg)
		case "withdraw_delegator_reward":
			return mapper.DistributionWithdrawDelegatorRewardToSub(msg, lf)
		case "fund_community_pool":
			return mapper.DistributionFundCommunityPoolToSub(msg)
		}
	case "evidence":
		switch msg.Type() {
		case "submit_evidence":
			return mapper.EvidenceSubmitEvidenceToSub(msg)
		}
	case "gov":
		switch msg.Type() {
		case "deposit":
			return mapper.GovDepositToSub(msg, lf)
		case "vote":
			return mapper.GovVoteToSub(msg)
		case "submit_proposal":
			return mapper.GovSubmitProposalToSub(msg, lf)
		}
	case "market":
		switch msg.Type() {
		case "swap":
			return mapper.MarketSwapToSub(msg, lf)
		case "swapsend":
			return mapper.MarketSwapSendToSub(msg, lf)
		}
	case "msgauth":
		switch msg.Type() {
		case "grant_authorization":
			return mapper.MsgauthGrantAuthorizationToSub(msg)
		case "revoke_authorization":
			return mapper.MsgauthRevokeAuthorizationToSub(msg)
		case "exec_delegated":
			se, msgs, er := mapper.MsgauthExecAuthorizedToSub(msg)
			if er != nil {
				return se, er
			}
			for _, subMsg := range msgs {
				subEv, subErr := getSubEvent(subMsg, lf)
				if subErr != nil {
					return se, err
				}
				se.Sub = append(se.Sub, subEv)

			}
			return se, nil
		}
	case "oracle":
		switch msg.Type() {
		case "exchangeratevote":
			return mapper.OracleExchangeRateVoteToSub(msg)
		case "exchangerateprevote":
			return mapper.OracleExchangeRatePrevoteToSub(msg)
		case "delegatefeeder":
			return mapper.OracleDelegateFeedConsent(msg)
		case "aggregateexchangerateprevote":
			return mapper.OracleAggregateExchangeRatePrevoteToSub(msg)
		case "aggregateexchangeratevote":
			return mapper.OracleAggregateExchangeRateVoteToSub(msg)
		}
	case "slashing":
		switch msg.Type() {
		case "unjail":
			return mapper.SlashingUnjailToSub(msg)
		}
	case "staking":
		switch msg.Type() {
		case "begin_unbonding":
			return mapper.StakingUndelegateToSub(msg, lf)
		case "edit_validator":
			return mapper.StakingEditValidatorToSub(msg)
		case "create_validator":
			return mapper.StakingCreateValidatorToSub(msg)
		case "delegate":
			return mapper.StakingDelegateToSub(msg, lf)
		case "begin_redelegate":
			return mapper.StakingBeginRedelegateToSub(msg, lf)
		}
	case "wasm":
		switch msg.Type() {
		case "execute_contract":
			return mapper.WasmExecuteContractToSub(msg)
		case "store_code":
			return mapper.WasmStoreCodeToSub(msg)
		case "update_contract_owner":
			return mapper.WasmMsgUpdateContractOwnerToSub(msg)
		case "instantiate_contract":
			return mapper.WasmMsgInstantiateContractToSub(msg)
		case "migrate_contract":
			return mapper.WasmMsgMigrateContractToSub(msg)
		}
	}

	return se, fmt.Errorf("unknown message %s, %s  ", msg.Route(), msg.Type())
}

type ToGet struct {
	Height  uint64
	Page    int
	PerPage int
}

func (c *Client) SingularHeightWorker(ctx context.Context, wg *sync.WaitGroup, out chan types.TxResponse, in chan ToGet) {
	defer wg.Done()

	for current := range in {
		resp, err := c.SearchTxSingularHeight(ctx, current.Height, current.Page, current.PerPage)
		if err != nil {
			c.logger.Error("[TERRA-API] Getting response from SearchTX", zap.Error(err), zap.Uint64("height", current.Height))
		}
		for _, r := range resp {
			out <- r
		}

		c.logger.Sync()
	}

}

// SearchTxSingularHeight is making search api call for
func (c *Client) SearchTxSingularHeight(ctx context.Context, height uint64, page, perPage int) (txSearch []types.TxResponse, err error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		return txSearch, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()

	s := strings.Builder{}
	s.WriteString(`"tx.height=`)
	s.WriteString(strconv.FormatUint(height, 10))
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	if c.rateLimiter != nil {
		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			return txSearch, err
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	if err != nil {
		return txSearch, err
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[TERRA-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		return txSearch, err
	}

	rawRequestHTTPDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &types.GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[TERRA-API] unable to decode result body", zap.Error(err))
		err = fmt.Errorf("unable to decode result body %w", err)
		return txSearch, err
	}

	if result.Error.Message != "" {
		c.logger.Error("[TERRA-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		return txSearch, err
	}

	if result.Result.TotalCount != "" {
		if err != nil {
			c.logger.Error("[TERRA-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery))
			return txSearch, err
		}
	}
	return result.Result.Txs, err
}

// GetFromRaw returns raw data for plugin use;
func (c *Client) GetFromRaw(logger *zap.Logger, txReader io.Reader) []map[string]interface{} {
	tx := &auth.StdTx{}
	base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)
	_, err := c.cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
	if err != nil {
		logger.Error("[TERRA-API] Problem decoding raw transaction (cdc) ", zap.Error(err))
	}
	slice := []map[string]interface{}{}
	for _, coin := range tx.Fee.Amount {
		slice = append(slice, map[string]interface{}{
			"text":     coin.Amount.String(),
			"numeric":  coin.Amount.BigInt(),
			"currency": coin.Denom,
		})
	}
	return slice
}

func findLog(lf []types.LogFormat, index int) types.LogFormat {
	if len(lf) <= index {
		return types.LogFormat{}
	}
	if l := lf[index]; l.MsgIndex == float64(index) {
		return l
	}
	for _, l := range lf {
		if l.MsgIndex == float64(index) {
			return l
		}
	}
	return types.LogFormat{}
}
