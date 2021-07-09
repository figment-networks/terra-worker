package client

import (
	"context"
	"fmt"

	"github.com/figment-networks/indexing-engine/structs"
	"go.uber.org/zap"
)

func (ic *IndexerClient) BlockAndTx(ctx context.Context, height uint64) (blockWM structs.BlockWithMeta, txsWM []structs.TransactionWithMeta, err error) {
	defer ic.logger.Sync()
	ic.logger.Debug("[TERRA-CLIENT] Getting height", zap.Uint64("block", height))

	hSess, err := ic.storeClient.GetSearchSession(ctx)
	if err != nil {
		return blockWM, nil, err
	}

	blockWM = structs.BlockWithMeta{Network: "terra", Version: "0.0.1"}
	blockWM.Block, err = ic.rpc.GetBlock(ctx, structs.HeightHash{Height: uint64(height)})
	blockWM.ChainID = blockWM.Block.ChainID
	if err != nil {
		ic.logger.Error("[TERRA-CLIENT] Err Getting block", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		return blockWM, nil, fmt.Errorf("error fetching block: %d %w ", uint64(height), err)
	}
	if err := hSess.StoreBlocks(ctx, []structs.BlockWithMeta{blockWM}); err != nil {
		return blockWM, nil, err
	}

	if blockWM.Block.NumberOfTransactions > 0 {
		ic.logger.Debug("[TERRA-CLIENT] Getting txs", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
		var txs []structs.Transaction
		txs, err = ic.rpc.SearchTx(ctx, structs.HeightHash{Height: height}, blockWM.Block, page)
		for _, t := range txs {
			txsWM = append(txsWM, structs.TransactionWithMeta{Network: "kava", ChainID: t.ChainID, Version: "0.0.1", Transaction: t})
		}
		if len(txsWM) > 0 {
			if err := hSess.StoreTransactions(ctx, txsWM); err != nil {
				return blockWM, txsWM, err
			}
		}
		ic.logger.Debug("[TERRA-CLIENT] txErr Getting txs", zap.Uint64("block", height), zap.Error(err), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	}

	if err := hSess.ConfirmHeights(ctx, []structs.BlockWithMeta{blockWM}); err != nil {
		return blockWM, txsWM, err
	}
	ic.logger.Debug("[TERRA-CLIENT] Got block", zap.Uint64("block", height), zap.Uint64("txs", blockWM.Block.NumberOfTransactions))
	return blockWM, txsWM, err
}
