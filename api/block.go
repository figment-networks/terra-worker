package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/figment-networks/indexer-manager/structs"
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

// GetBlocksMeta fetches block metadata from given range of blocks
func (c Client) GetBlocksMeta(ctx context.Context, params structs.HeightRange, limit uint64, blocks *BlocksMap, end chan<- error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/blockchain", nil)
	if err != nil {
		end <- err
		return
	}

	req.Header.Add("Content-Type", "application/json")
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
	rawRequestDuration.WithLabels("/blockchain", resp.Status).Observe(time.Since(n).Seconds())
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockchainResponse
	if err = decoder.Decode(&result); err != nil {
		end <- err
		return
	}

	if result.Error.Message != "" {
		end <- fmt.Errorf("error fetching block: %s ", result.Error.Message)
		return
	}

	blocks.Lock()
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
	blocks.Unlock()

	end <- nil
	return
}
