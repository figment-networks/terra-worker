package api

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/figment-networks/indexing-engine/structs"
	"github.com/tendermint/tendermint/libs/bytes"
	"google.golang.org/grpc"
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

// GetBlock fetches most recent block from chain
func (c *Client) GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, er error) {
	var ok bool
	if params.Height != 0 {
		block, ok = c.Sbc.Get(params.Height)
		if ok {
			return block, nil
		}
	}

	if err := c.rateLimiterGRPC.Wait(ctx); err != nil {
		return block, err
	}

	nctx, cancel := context.WithTimeout(ctx, c.cfg.TimeoutBlockCall)
	defer cancel()
	n := time.Now()
	if params.Height == 0 {
		lb, err := c.tmServiceClient.GetLatestBlock(nctx, &tmservice.GetLatestBlockRequest{})
		if err != nil {
			rawRequestGRPCDuration.WithLabels("GetLatestBlock", "error").Observe(time.Since(n).Seconds())
			return block, err
		}
		rawRequestGRPCDuration.WithLabels("GetLatestBlock", "ok").Observe(time.Since(n).Seconds())

		bh := bytes.HexBytes(lb.BlockId.Hash)

		block = structs.Block{
			Hash:                 bh.String(),
			Height:               uint64(lb.Block.Header.Height),
			Time:                 lb.Block.Header.Time,
			ChainID:              lb.Block.Header.ChainID,
			NumberOfTransactions: uint64(len(lb.Block.Data.Txs)),
		}
		c.Sbc.Add(block)

		return block, nil
	}

	bbh, err := c.tmServiceClient.GetBlockByHeight(nctx, &tmservice.GetBlockByHeightRequest{Height: int64(params.Height)}, grpc.WaitForReady(true))
	if err != nil {
		rawRequestGRPCDuration.WithLabels("GetBlockByHeight", "error").Observe(time.Since(n).Seconds())
		return block, err
	}
	rawRequestGRPCDuration.WithLabels("GetBlockByHeight", "ok").Observe(time.Since(n).Seconds())

	hb := bytes.HexBytes(bbh.BlockId.Hash)
	block = structs.Block{
		Hash:                 hb.String(),
		Height:               uint64(bbh.Block.Header.Height),
		Time:                 bbh.Block.Header.Time,
		ChainID:              bbh.Block.Header.ChainID,
		NumberOfTransactions: uint64(len(bbh.Block.Data.Txs)),
	}

	c.Sbc.Add(block)

	return block, nil
}
