package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

	decoder := json.NewDecoder(resp.Body)

	var result *types.GetBlockResponseV4 // todo v3
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

	return block, nil
}

// GetBlocksMeta fetches block metadata from given range of blocks
func (c Client) GetBlocksMeta(ctx context.Context, params structs.HeightRange, limit uint64, blocks *BlocksMap, end chan<- error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/blockchain", nil)
	if err != nil {
		end <- err
		return
	}

	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	if params.StartHeight > 0 {
		q.Add("minHeight", strconv.FormatUint(params.StartHeight, 10))
	}

	if params.EndHeight > 0 {
		q.Add("maxHeight", strconv.FormatUint(params.EndHeight, 10))
	}

	if limit > 0 {
		q.Add("limit", strconv.FormatUint(limit, 10))
	}

	req.URL.RawQuery = q.Encode()

	if c.rateLimiter != nil {
		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			end <- err
			return
		}
	}

	n := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		end <- err
		return
	}

	rawRequestHTTPDuration.WithLabels("/blockchain", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	if resp.StatusCode != 200 { // (lukanus): for Datahub errors
		allBody, _ := ioutil.ReadAll(resp.Body)
		end <- fmt.Errorf("Bad Response %d (%s)", resp.StatusCode, string(allBody))
		return
	}

	if params.ChainID == "columbus-4" {
		if err := decodeBlocksColumbus4(resp.Body, blocks); err != nil {
			end <- err
			return
		}
	} else {
		if err := decodeBlocksColumbus3(resp.Body, blocks); err != nil {
			end <- err
			return
		}
	}

	end <- nil
	return
}

func decodeBlocksColumbus3(respBody io.ReadCloser, blocks *BlocksMap) (err error) {
	decoder := json.NewDecoder(respBody)

	var result *types.GetBlockchainResponse
	if err := decoder.Decode(&result); err != nil {
		return err
	}

	if result.Error.Message != "" {
		return fmt.Errorf("error fetching block: %s ", result.Error.Message)
	}

	blocks.Lock()
	defer blocks.Unlock()
	for _, meta := range result.Result.BlockMetas {

		bTime, _ := time.Parse(time.RFC3339Nano, meta.Header.Time)
		uHeight, _ := strconv.ParseUint(meta.Header.Height, 10, 64)
		numTxs, _ := strconv.ParseUint(meta.Header.NumTxs, 10, 64)

		block := structs.Block{
			Hash:                 meta.BlockID.Hash,
			Height:               uHeight,
			ChainID:              meta.Header.ChainID,
			Time:                 bTime,
			NumberOfTransactions: numTxs,
		}
		blocks.NumTxs += numTxs
		if blocks.StartHeight == 0 || blocks.StartHeight > block.Height {
			blocks.StartHeight = block.Height
		}
		if blocks.EndHeight == 0 || blocks.EndHeight < block.Height {
			blocks.EndHeight = block.Height
		}
		blocks.Blocks[block.Height] = block
	}

	return
}

func decodeBlocksColumbus4(respBody io.ReadCloser, blocks *BlocksMap) (err error) {
	decoder := json.NewDecoder(respBody)

	var result *types.GetBlockchainResponseV4
	if err := decoder.Decode(&result); err != nil {
		return err
	}

	if result.Error.Message != "" {
		return fmt.Errorf("error fetching block: %s ", result.Error.Message)
	}

	blocks.Lock()
	defer blocks.Unlock()
	for _, meta := range result.Result.BlockMetas {

		bTime, _ := time.Parse(time.RFC3339Nano, meta.Header.Time)
		uHeight, _ := strconv.ParseUint(meta.Header.Height, 10, 64)
		numTxs, _ := strconv.ParseUint(meta.NumTxs, 10, 64)

		block := structs.Block{
			Hash:                 meta.BlockID.Hash,
			Height:               uHeight,
			ChainID:              meta.Header.ChainID,
			Time:                 bTime,
			NumberOfTransactions: numTxs,
		}
		blocks.NumTxs += numTxs
		if blocks.StartHeight == 0 || blocks.StartHeight > block.Height {
			blocks.StartHeight = block.Height
		}
		if blocks.EndHeight == 0 || blocks.EndHeight < block.Height {
			blocks.EndHeight = block.Height
		}

		blocks.Blocks[block.Height] = block
	}
	return
}
