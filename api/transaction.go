package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/figment-networks/indexer-manager/structs"
	cStruct "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"

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

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-]+)[\\s]*([^0-9\\s]+)$")

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

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &ResultTxSearch{}
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

	totalCount := int64(0)
	if result.TotalCount != "" {
		totalCount, err = strconv.ParseInt(result.TotalCount, 10, 64)
		if err != nil {
			c.logger.Error("[TERRA-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
			fin <- err.Error()
			return
		}
	}
	numberOfItemsTransactions.Observe(float64(totalCount))
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

func RawToTransaction(logger *zap.Logger, cdc *amino.Codec, in []TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp) error {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for _, txRaw := range in {
		readr.Reset(txRaw.TxResult.Log)
		lf := []LogFormat{}
		txErr := TxLogError{}
		err := dec.Decode(&lf)
		if err != nil {
			if err != nil {
				// (lukanus): Try to fallback to known error format
				txErr.Message = txRaw.TxResult.Log
			}
		}

		tx, err := rawToTransaction(logger, cdc, txRaw, lf, txErr, blocks)
		if err != nil {
			return err
		}
		out <- tx
	}

	return nil
}

func RawToTransactionCh(logger *zap.Logger, cdc *amino.Codec, wg *sync.WaitGroup, in <-chan TxResponse, blocks map[uint64]structs.Block, out chan cStruct.OutResp) {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	defer wg.Done()
	for txRaw := range in {
		lf := []LogFormat{}
		txErr := TxLogError{}
		if txRaw.TxResult.Log != "" {
			readr.Reset(txRaw.TxResult.Log)

			err := dec.Decode(&lf)
			if err != nil {
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

func rawToTransaction(logger *zap.Logger, cdc *amino.Codec, txRaw TxResponse, txLog []LogFormat, txErr TxLogError, blocks map[uint64]structs.Block) (cStruct.OutResp, error) {
	timer := metrics.NewTimer(transactionConversionDuration)
	defer timer.ObserveDuration()

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

		ev, err := getSubEvent(msg)
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

func eventFromLogs(lf LogFormat) structs.TransactionEvent {

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

func getCurrency(in string) []string {
	return curencyRegex.FindStringSubmatch(in)
}

func getSubEvent(msg sdk.Msg) (se structs.SubsetEvent, err error) {
	switch msg.Route() {
	case "bank":
		switch msg.Type() {
		case "multisend":
			return mapBankMultisendToSub(msg)
		case "send":
			return mapBankSendToSub(msg)
		}
	case "crisis":
		switch msg.Type() {
		case "verify_invariant":
			return mapCrisisVerifyInvariantToSub(msg)
		}
	case "distribution":
		switch msg.Type() {
		case "withdraw_validator_commission":
			return mapDistributionWithdrawValidatorCommissionToSub(msg)
		case "set_withdraw_address":
			return mapDistributionSetWithdrawAddressToSub(msg)
		case "withdraw_delegator_reward":
			return mapDistributionWithdrawDelegatorRewardToSub(msg)
		case "fund_community_pool":
			return mapDistributionFundCommunityPoolToSub(msg)
		}
	case "evidence":
		switch msg.Type() {
		case "submit_evidence":
			return mapEvidenceSubmitEvidenceToSub(msg)
		}
	case "gov":
		switch msg.Type() {
		case "deposit":
			return mapGovDepositToSub(msg)
		case "vote":
			return mapGovVoteToSub(msg)
		case "submit_proposal":
			return mapGovSubmitProposalToSub(msg)
		}
	case "market":
		switch msg.Type() {
		case "swap":
			return mapMarketSwapToSub(msg)
		case "swapsend":
			return mapMarketSwapSendToSub(msg)
		}
	case "msgauth":
		switch msg.Type() {
		case "grant_authorization":
			return mapMsgauthGrantAuthorizationToSub(msg)
		case "revoke_authorization":
			return mapMsgauthRevokeAuthorizationToSub(msg)
		case "exec_delegated":
			se, msgs, er := mapMsgauthExecAuthorizedToSub(msg)
			if er != nil {
				return se, er
			}
			for _, subMsg := range msgs {
				subEv, subErr := getSubEvent(subMsg)
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
			return mapOracleExchangeRateVoteToSub(msg)
		case "exchangerateprevote":
			return mapOracleExchangeRatePrevoteToSub(msg)
		case "delegatefeeder":
			return mapOracleDelegateFeedConsent(msg)
		case "aggregateexchangerateprevote":
			return mapOracleAggregateExchangeRatePrevoteToSub(msg)
		case "aggregateexchangeratevote":
			return mapOracleAggregateExchangeRateVoteToSub(msg)
		}
	case "slashing":
		switch msg.Type() {
		case "unjail":
			return mapSlashingUnjailToSub(msg)
		}
	case "staking":
		switch msg.Type() {
		case "begin_unbonding":
			return mapStakingUndelegateToSub(msg)
		case "edit_validator":
			return mapStakingEditValidatorToSub(msg)
		case "create_validator":
			return mapStakingCreateValidatorToSub(msg)
		case "delegate":
			return mapStakingDelegateToSub(msg)
		case "begin_redelegate":
			return mapStakingBeginRedelegateToSub(msg)
		}
	case "wasm":
		switch msg.Type() {
		case "execute_contract":
			return mapWasmExecuteContractToSub(msg)
		case "store_code":
			return mapWasmStoreCodeToSub(msg)
		case "update_contract_owner":
			return mapWasmMsgUpdateContractOwnerToSub(msg)
		case "instantiate_contract":
			return mapWasmMsgInstantiateContractToSub(msg)
		case "migrate_contract":
			return mapWasmMsgMigrateContractToSub(msg)
		}
	}

	return se, fmt.Errorf("unknown message %s, %s  ", msg.Route(), msg.Type())
}

type ToGet struct {
	Height  uint64
	Page    int
	PerPage int
}

func (c *Client) SingularHeightWorker(ctx context.Context, wg *sync.WaitGroup, out chan TxResponse, in chan ToGet) {
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
func (c *Client) SearchTxSingularHeight(ctx context.Context, height uint64, page, perPage int) (txSearch []TxResponse, err error) {
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

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
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

	totalCount := int64(0)
	if result.Result.TotalCount != "" {
		totalCount, err = strconv.ParseInt(result.Result.TotalCount, 10, 64)
		if err != nil {
			c.logger.Error("[TERRA-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery))
			return txSearch, err
		}
	}
	numberOfItemsTransactions.Observe(float64(totalCount))
	return result.Result.Txs, err
}

func getCoin(s string) (number *big.Int, exp int32, err error) {
	s = strings.Replace(s, ",", ".", -1)
	strs := strings.Split(s, `.`)
	number = &big.Int{}
	if len(strs) == 1 {
		number.SetString(strs[0], 10)
		return number, 0, nil
	}
	if len(strs) == 2 {
		number.SetString(strs[0]+strs[1], 10)
		return number, int32(len(strs[1])), nil
	}

	return number, 0, errors.New("Impossible to parse ")
}
