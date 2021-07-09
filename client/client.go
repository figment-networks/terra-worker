package client

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	mStructs "github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/indexing-engine/worker/process/ranged"
	"github.com/figment-networks/indexing-engine/worker/store"
	"github.com/figment-networks/terra-worker/api"
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
	GetBlock(ctx context.Context, params structs.HeightHash) (block structs.Block, err error)
	SearchTx(ctx context.Context, r structs.HeightHash, block structs.Block, perPage uint64) (txs []structs.Transaction, err error)
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

	Reqester            *ranged.RangeRequester
	storeClient         store.SearchStoreCaller
	maximumHeightsToGet uint64
}

func NewIndexerClient(ctx context.Context, logger *zap.Logger, lcdCli LCD, rpcCli RPC, storeClient store.SearchStoreCaller, maximumHeightsToGet uint64) *IndexerClient {
	getTransactionDuration = endpointDuration.WithLabels("getTransactions")
	getLatestDuration = endpointDuration.WithLabels("getLatest")
	getBlockDuration = endpointDuration.WithLabels("getBlock")
	getAccountBalanceDuration = endpointDuration.WithLabels("getAccountBalance")
	getAccountDelegationsDuration = endpointDuration.WithLabels("getAccountDelegations")
	api.InitMetrics()

	ic := &IndexerClient{
		rpc:                 rpcCli,
		lcd:                 lcdCli,
		logger:              logger,
		storeClient:         storeClient,
		maximumHeightsToGet: maximumHeightsToGet,
		streams:             make(map[uuid.UUID]*cStructs.StreamAccess),
	}

	ic.Reqester = ranged.NewRangeRequester(ic, 20)
	return ic
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
			case mStructs.ReqIDGetTransactions:
				ic.GetTransactions(nCtx, taskRequest, stream, ic.rpc)
			case mStructs.ReqIDGetLatestMark:
				ic.GetLatestMark(nCtx, taskRequest, stream, ic.rpc)
			case mStructs.ReqIDGetReward:
				ic.GetReward(nCtx, taskRequest, stream, ic.lcd)
			case mStructs.ReqIDAccountBalance:
				ic.GetAccountBalance(nCtx, taskRequest, stream, ic.lcd)
			case mStructs.ReqIDAccountDelegations:
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

// GetTransactions gets new transactions and blocks from terra for given range
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

	ic.logger.Debug("[TERRA-CLIENT] Getting Range", zap.Stringer("taskID", tr.Id), zap.Uint64("start", hr.StartHeight), zap.Uint64("end", hr.EndHeight))

	heights, err := ic.Reqester.GetRange(ctx, *hr)
	resp := &cStructs.TaskResponse{
		Id:    tr.Id,
		Type:  "Heights",
		Final: true,
	}
	if heights.NumberOfHeights > 0 {
		resp.Payload, _ = json.Marshal(heights)
	}
	if err != nil {
		resp.Error = cStructs.TaskError{Msg: err.Error()}
		ic.logger.Error("[TERRA-CLIENT] Error getting range (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		if err := stream.Send(*resp); err != nil {
			ic.logger.Error("[TERRA-CLIENT] Error sending message (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
		}
		return
	}
	if err := stream.Send(*resp); err != nil {
		ic.logger.Error("[TERRA-CLIENT] Error sending message (Get Transactions) ", zap.Error(err), zap.Stringer("taskID", tr.Id))
	}
	ic.logger.Debug("[TERRA-CLIENT] Finished sending all", zap.Stringer("taskID", tr.Id), zap.Any("heights", hr))
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
