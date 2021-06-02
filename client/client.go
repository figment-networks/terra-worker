package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tendermint/go-amino"
	"go.uber.org/zap"

	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/terra-worker/api"
	"github.com/figment-networks/terra-worker/api/types"
)

//go:generate mockgen -destination=./mocks/mock_client.go -package=mocks -imports github.com/tendermint/go-amino github.com/figment-networks/terra-worker/client RPC

const page = 100
const blockchainEndpointLimit = 20

var (
	getTransactionDuration        *metrics.GroupObserver
	getLatestDuration             *metrics.GroupObserver
	getBlockDuration              *metrics.GroupObserver
	getAccountBalanceDuration     *metrics.GroupObserver
	getAccountDelegationsDuration *metrics.GroupObserver
)

type RPC interface {
	CDC() *amino.Codec
	SingularHeightWorker(ctx context.Context, wg *sync.WaitGroup, out chan types.TxResponse, in chan api.ToGet)
	GetBlocksMeta(ctx context.Context, params structs.HeightRange, limit uint64, blocks *api.BlocksMap, end chan<- error)
}

type LCD interface {
	GetReward(ctx context.Context, params structs.HeightAccount) (resp structs.GetRewardResponse, err error)
	GetAccountBalance(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountBalanceResponse, err error)
	GetAccountDelegations(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountDelegationsResponse, err error)
}

type IndexerClient struct {
	lcd LCD
	rpc RPC

	logger  *zap.Logger
	streams map[uuid.UUID]*cStructs.StreamAccess
	sLock   sync.Mutex

	bigPage             uint64
	maximumHeightsToGet uint64
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, lcdCli LCD, rpcCli RPC, bigPage, maximumHeightsToGet uint64) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")
	getAccountBalanceDuration = endpointDuration.WithLabels("getAccountBalance")
	getAccountDelegationsDuration = endpointDuration.WithLabels("getAccountDelegations")

	return &IndexerClient{
		rpc:                 rpcCli,
		lcd:                 lcdCli,
		logger:              logger,
		bigPage:             bigPage,
		maximumHeightsToGet: maximumHeightsToGet,
		streams:             make(map[uuid.UUID]*cStructs.StreamAccess),
	}
}

// CloseStream removes stream from worker/client
func (ic *IndexerClient) CloseStream(ctx context.Context, streamID uuid.UUID) error {
	ic.sLock.Lock()
	defer ic.sLock.Unlock()

	ic.logger.Debug("[TERRA-CLIENT] Close Stream", zap.Stringer("streamID", streamID))
	delete(ic.streams, streamID)

	return nil
}

// RegisterStream adds new listeners to the streams - currently fixed number per stream
func (ic *IndexerClient) RegisterStream(ctx context.Context, stream *cStructs.StreamAccess) error {
	ic.logger.Debug("[TERRA-CLIENT] Register Stream", zap.Stringer("streamID", stream.StreamID))
	newStreamsMetric.WithLabels().Inc()

	ic.sLock.Lock()
	defer ic.sLock.Unlock()
	ic.streams[stream.StreamID] = stream

	// Limit workers not to create new goroutines over and over again
	for i := 0; i < 20; i++ {
		go ic.Run(ctx, stream)
	}

	return nil
}

func (ic *IndexerClient) Run(ctx context.Context, stream *cStructs.StreamAccess) {

	for {
		select {
		case <-ctx.Done():
			ic.sLock.Lock()
			delete(ic.streams, stream.StreamID)
			ic.sLock.Unlock()
			return
		case <-stream.Finish:
			return
		case taskRequest := <-stream.RequestListener:
			receivedRequestsMetric.WithLabels(taskRequest.Type).Inc()
			nCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			switch taskRequest.Type {
			case structs.ReqIDGetTransactions:
				ic.GetTransactions(nCtx, taskRequest, stream, ic.rpc)
			case structs.ReqIDLatestData:
				ic.GetLatest(nCtx, taskRequest, stream, ic.rpc)
			case structs.ReqIDGetReward:
				ic.GetReward(nCtx, taskRequest, stream, ic.lcd)
			case structs.ReqIDAccountBalance:
				ic.GetAccountBalance(nCtx, taskRequest, stream, ic.lcd)
			case structs.ReqIDAccountDelegations:
				ic.GetAccountDelegations(nCtx, taskRequest, stream, ic.lcd)
			default:
				stream.Send(cStructs.TaskResponse{
					Id:    taskRequest.Id,
					Error: cStructs.TaskError{Msg: "There is no such handler " + taskRequest.Type},
					Final: true,
				})
			}
			cancel()
		}
	}

}

func (ic *IndexerClient) GetTransactions(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client RPC) {
	timer := metrics.NewTimer(getTransactionDuration)
	defer timer.ObserveDuration()

	hr := &structs.HeightRange{}
	err := json.Unmarshal(tr.Payload, hr)
	if err != nil {
		ic.logger.Debug("[TERRA-CLIENT] Cannot unmarshal payload", zap.String("contents", string(tr.Payload)))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "cannot unmarshal payload: " + err.Error()},
			Final: true,
		})
		return
	}

	if hr.EndHeight == 0 {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "end height is zero" + err.Error()},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make(chan cStructs.OutResp, page*2+1)
	fin := make(chan bool, 2)

	go sendResp(sCtx, tr.Id, out, ic.logger, stream, fin)

	var i uint64
	for {
		hrInner := structs.HeightRange{
			StartHeight: hr.StartHeight + i*uint64(ic.bigPage),
			EndHeight:   hr.StartHeight + i*uint64(ic.bigPage) + uint64(ic.bigPage) - 1,
			Network:     hr.Network,
			ChainID:     hr.ChainID,
		}
		if hrInner.EndHeight > hr.EndHeight {
			hrInner.EndHeight = hr.EndHeight
		}

		if err := getRangeSingular(sCtx, ic.logger, client, hrInner, out); err != nil {
			stream.Send(cStructs.TaskResponse{
				Id:    tr.Id,
				Error: cStructs.TaskError{Msg: err.Error()},
				Final: true,
			})
			ic.logger.Error("[TERRA-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
			return
		}
		i++
		if hrInner.EndHeight == hr.EndHeight {
			break
		}
	}

	ic.logger.Debug("[TERRA-CLIENT] Received all", zap.Stringer("taskID", tr.Id))
	close(out)

	for {
		select {
		case <-sCtx.Done():
			return
		case <-fin:
			ic.logger.Debug("[TERRA-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id))
			return
		}
	}
}

// sendResp sends responses to out channel preparing
func sendResp(ctx context.Context, id uuid.UUID, out chan cStructs.OutResp, logger *zap.Logger, stream *cStructs.StreamAccess, fin chan bool) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	order := uint64(0)

	var contextDone bool

SendLoop:
	for {
		select {
		case <-ctx.Done():
			contextDone = true
			break SendLoop
		case t, ok := <-out:
			if !ok && t.Type == "" {
				break SendLoop
			}
			b.Reset()

			err := enc.Encode(t.Payload)
			if err != nil {
				logger.Error("[TERRA-CLIENT] Error encoding payload data", zap.Error(err))
			}

			tr := cStructs.TaskResponse{
				Id:      id,
				Type:    t.Type,
				Order:   order,
				Payload: make([]byte, b.Len()),
			}

			b.Read(tr.Payload)
			order++
			err = stream.Send(tr)
			if err != nil {
				logger.Error("[TERRA-CLIENT] Error sending data", zap.Error(err))
			}
			sendResponseMetric.WithLabels(t.Type, "yes").Inc()
		}
	}

	err := stream.Send(cStructs.TaskResponse{
		Id:    id,
		Type:  "END",
		Order: order,
		Final: true,
	})

	if err != nil {
		logger.Error("[TERRA-CLIENT] Error sending end", zap.Error(err))
	}

	if fin != nil {
		if !contextDone {
			fin <- true
		}
		close(fin)
	}

}

// GetLatest gets latest transactions and blocks.
// It gets latest transaction, then diff it with
func (ic *IndexerClient) GetLatest(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client RPC) {
	timer := metrics.NewTimer(getLatestDuration)
	defer timer.ObserveDuration()

	ldr := &structs.LatestDataRequest{}
	err := json.Unmarshal(tr.Payload, ldr)
	if err != nil {
		stream.Send(cStructs.TaskResponse{Id: tr.Id, Error: cStructs.TaskError{Msg: "Cannot unmarshal payment"}, Final: true})
	}

	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	batchesCtrl := make(chan error, 2)
	defer close(batchesCtrl)
	blocksAll := &api.BlocksMap{Blocks: map[uint64]structs.Block{}}

	// get latest blocks
	client.GetBlocksMeta(sCtx, structs.HeightRange{
		Network: ldr.Network,
		ChainID: ldr.ChainID,
	}, blockchainEndpointLimit, blocksAll, batchesCtrl)

	if err := <-batchesCtrl; err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: err.Error()},
			Final: true,
		})
		return
	}

	startingHeight := getStartingHeight(ldr.LastHeight, ic.maximumHeightsToGet, blocksAll.EndHeight)
	if startingHeight <= blocksAll.StartHeight {
		var i, responses uint64
		for {
			bhr := structs.HeightRange{
				StartHeight: startingHeight + i*uint64(blockchainEndpointLimit),
				EndHeight:   startingHeight + i*uint64(blockchainEndpointLimit) + uint64(blockchainEndpointLimit) - 1,
				Network:     ldr.Network,
				ChainID:     ldr.ChainID,
			}

			if bhr.EndHeight > blocksAll.StartHeight {
				bhr.EndHeight = blocksAll.StartHeight
			}

			ic.logger.Debug("[TERRA-CLIENT] Getting blocks for ", zap.Uint64("end", bhr.EndHeight), zap.Uint64("start", bhr.StartHeight))
			go client.GetBlocksMeta(ctx, bhr, blockchainEndpointLimit, blocksAll, batchesCtrl)
			i++
			if bhr.EndHeight == blocksAll.StartHeight {
				break
			}
		}

		var errors = []error{}
		for err := range batchesCtrl {
			responses++
			if err != nil {
				errors = append(errors, err)
			}
			if responses == i {
				break
			}
		}

		if len(errors) > 0 {
			errString := ""
			for _, err := range errors {
				errString += err.Error() + " , "
			}

			stream.Send(cStructs.TaskResponse{
				Id:    tr.Id,
				Error: cStructs.TaskError{Msg: fmt.Sprintf("Errors Getting Blocks: - %s ", errString)},
				Final: true,
			})
			return
		}
	}

	out := make(chan cStructs.OutResp, page)
	fin := make(chan bool, 2)
	// (lukanus): in separate goroutine take transaction format wrap it in transport message and send
	go sendResp(sCtx, tr.Id, out, ic.logger, stream, fin)

	convertWG := &sync.WaitGroup{}
	txIn := make(chan types.TxResponse, 20)
	convertWG.Add(1)
	go api.RawToTransactionCh(ic.logger, client.CDC(), convertWG, txIn, blocksAll.Blocks, out)

	httpReqWG := &sync.WaitGroup{}
	toGet := make(chan api.ToGet, 10)
	for i := 0; i < 5; i++ {
		httpReqWG.Add(1)
		go client.SingularHeightWorker(ctx, httpReqWG, txIn, toGet)
	}

	for h, block := range blocksAll.Blocks {
		// (lukanus): skip processing blocks before given range
		// we take blocks by 20
		if blocksAll.StartHeight < block.Height {
			continue
		}

		out <- cStructs.OutResp{
			Type:    "Block",
			Payload: block,
		}

		if block.NumberOfTransactions > 0 {
			toBeDone := int(math.Ceil(float64(block.NumberOfTransactions) / float64(page)))
			for i := 0; i < toBeDone; i++ {
				toGet <- api.ToGet{
					Height:  h,
					Page:    i + 1,
					PerPage: page,
				}
			}
		}
	}

	close(toGet)
	httpReqWG.Wait()
	close(txIn)
	convertWG.Wait()

	ic.logger.Debug("[TERRA-CLIENT] Received all", zap.Stringer("taskID", tr.Id))
	close(out)

}

// GetReward gets reward
func (ic *IndexerClient) GetReward(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client LCD) {
	timer := metrics.NewTimer(getBlockDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	reward, err := client.GetReward(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting reward", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting reward data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "Reward",
		Payload: reward,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// GetAccountBalance gets account balance
func (ic *IndexerClient) GetAccountBalance(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client LCD) {
	timer := metrics.NewTimer(getAccountBalanceDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	blnc, err := client.GetAccountBalance(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting account balance", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting account balance data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "AccountBalance",
		Payload: blnc,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// GetAccountDelegations gets account delegations
func (ic *IndexerClient) GetAccountDelegations(ctx context.Context, tr cStructs.TaskRequest, stream *cStructs.StreamAccess, client LCD) {
	timer := metrics.NewTimer(getAccountDelegationsDuration)
	defer timer.ObserveDuration()

	ha := &structs.HeightAccount{}
	err := json.Unmarshal(tr.Payload, ha)
	if err != nil {
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Cannot unmarshal payload"},
			Final: true,
		})
		return
	}

	sCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	blnc, err := client.GetAccountDelegations(sCtx, *ha)
	if err != nil {
		ic.logger.Error("Error getting account balance", zap.Error(err))
		stream.Send(cStructs.TaskResponse{
			Id:    tr.Id,
			Error: cStructs.TaskError{Msg: "Error getting account balance data " + err.Error()},
			Final: true,
		})
		return
	}

	out := make(chan cStructs.OutResp, 1)
	out <- cStructs.OutResp{
		ID:      tr.Id,
		Type:    "AccountDelegations",
		Payload: blnc,
	}
	close(out)

	sendResp(ctx, tr.Id, out, ic.logger, stream, nil)
}

// getRange gets given range of blocks and transactions
func getRangeSingular(ctx context.Context, logger *zap.Logger, client RPC, hr structs.HeightRange, out chan cStructs.OutResp) error {
	defer logger.Sync()

	batchesCtrl := make(chan error, 2)
	defer close(batchesCtrl)
	blocksAll := &api.BlocksMap{Blocks: map[uint64]structs.Block{}}

	var i uint64
	for {
		bhr := structs.HeightRange{
			StartHeight: hr.StartHeight + i*uint64(blockchainEndpointLimit),
			EndHeight:   hr.StartHeight + i*uint64(blockchainEndpointLimit) + uint64(blockchainEndpointLimit) - 1,
			Network:     hr.Network,
			ChainID:     hr.ChainID,
		}

		if bhr.EndHeight > hr.EndHeight {
			bhr.EndHeight = hr.EndHeight
		}

		logger.Debug("[TERRA-CLIENT] Getting blocks for ", zap.Uint64("end", bhr.EndHeight), zap.Uint64("start", bhr.StartHeight))
		go client.GetBlocksMeta(ctx, bhr, 0, blocksAll, batchesCtrl)
		i++
		if bhr.EndHeight == hr.EndHeight {
			break
		}
	}

	var responses uint64
	var errors = []error{}
	for err := range batchesCtrl {
		responses++
		if err != nil {
			errors = append(errors, err)
		}
		if responses == i {
			break
		}
	}

	if len(errors) > 0 {
		errString := ""
		for _, err := range errors {
			errString += err.Error() + " , "
		}
		return fmt.Errorf("Errors Getting Blocks: - %s ", errString)
	}

	convertWG := &sync.WaitGroup{}
	txIn := make(chan types.TxResponse, 20)
	convertWG.Add(1)
	go api.RawToTransactionCh(logger, client.CDC(), convertWG, txIn, blocksAll.Blocks, out)

	httpReqWG := &sync.WaitGroup{}
	toGet := make(chan api.ToGet, 10)
	for i := 0; i < 5; i++ {
		httpReqWG.Add(1)
		go client.SingularHeightWorker(ctx, httpReqWG, txIn, toGet)
	}

	for h, block := range blocksAll.Blocks {
		out <- cStructs.OutResp{
			Type:    "Block",
			Payload: block,
		}

		if block.NumberOfTransactions > 0 {
			toBeDone := int(math.Ceil(float64(block.NumberOfTransactions) / float64(page)))
			for i := 0; i < toBeDone; i++ {
				toGet <- api.ToGet{
					Height:  h,
					Page:    i + 1,
					PerPage: page,
				}
			}
		}
	}

	close(toGet)
	httpReqWG.Wait()
	close(txIn)
	convertWG.Wait()

	return nil
}

// getStartingHeight - based current state
func getStartingHeight(lastHeight, maximumHeightsToGet, blockHeightFromChain uint64) (startingHeight uint64) {
	// (lukanus): When nothing is scraped we want to get only X number of last requests
	if lastHeight == 0 {
		lastX := blockHeightFromChain - maximumHeightsToGet
		if lastX > 0 {
			return lastX
		}
	}

	if maximumHeightsToGet < blockHeightFromChain-lastHeight {
		if maximumHeightsToGet > blockHeightFromChain {
			return 0
		}
		return blockHeightFromChain - maximumHeightsToGet
	}

	return lastHeight
}
