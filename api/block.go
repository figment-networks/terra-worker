package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/types"
)

// BlocksMap map of blocks to control block map
// with extra summary of number of transactions
type BlocksMap struct {
	sync.Mutex
	Blocks      map[uint64]structs.Block
	NumTxs      uint64
	StartHeight uint64
	EndHeight   uint64
}

// GetBlock fetches block from chain.
func (c Client) GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, err error) {
	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return block, err
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	req, err := http.NewRequestWithContext(sCtx, http.MethodGet, c.baseURL+"/block", nil)
	if err != nil {
		return block, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatUint(params.Height, 10))
	}
	req.URL.RawQuery = q.Encode()

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return block, err
	}
	rawRequestHTTPDuration.WithLabels("/block", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	if c.chainID == "columbus-4" {
		return decodeBlockColumbus4(resp.Body)
	}
	return decodeBlockColumbus3(resp.Body)
}

func decodeBlockColumbus4(respBody io.ReadCloser) (block structs.Block, err error) {
	decoder := json.NewDecoder(respBody)

	var result *types.GetBlockResponseV4
	if err = decoder.Decode(&result); err != nil {
		return block, err
	}

	if result.Error.Message != "" {
		return block, fmt.Errorf("[TERRA-API] Error fetching block: %s ", result.Error.Message)
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	if err != nil {
		return block, err
	}
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)
	if err != nil {
		return block, err
	}

	numTxs := len(result.Result.Block.Data.Txs)

	block = structs.Block{
		Hash:                 result.Result.BlockID.Hash,
		Height:               uHeight,
		Time:                 bTime,
		ChainID:              result.Result.Block.Header.ChainID,
		NumberOfTransactions: uint64(numTxs),
	}
	return
}

func decodeBlockColumbus3(respBody io.ReadCloser) (block structs.Block, err error) {
	decoder := json.NewDecoder(respBody)

	var result *types.GetBlockResponse
	if err = decoder.Decode(&result); err != nil {
		return block, err
	}

	if result.Error.Message != "" {
		return block, fmt.Errorf("[TERRA-API] Error fetching block: %s ", result.Error.Message)
	}
	bTime, err := time.Parse(time.RFC3339Nano, result.Result.Block.Header.Time)
	if err != nil {
		return block, err
	}
	uHeight, err := strconv.ParseUint(result.Result.Block.Header.Height, 10, 64)
	if err != nil {
		return block, err
	}

	numTxs, err := strconv.ParseUint(result.Result.Block.Header.NumTxs, 10, 64)
	if err != nil {
		return block, err
	}

	block = structs.Block{
		Hash:                 result.Result.BlockMeta.BlockID.Hash,
		Height:               uHeight,
		Time:                 bTime,
		ChainID:              result.Result.Block.Header.ChainID,
		NumberOfTransactions: numTxs,
	}

	return
}
