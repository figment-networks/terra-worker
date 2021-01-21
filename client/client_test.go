package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/terra-worker/api"
	"github.com/figment-networks/terra-worker/api/types"
	apiMocks "github.com/figment-networks/terra-worker/client/mocks"

	"github.com/golang/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func TestIndexerClient_GetTransactions(t *testing.T) {
	var bigPage uint64 = 30
	var maximumHeightsToGet uint64 = 30

	type args struct {
		ctx    context.Context
		stream *cStructs.StreamAccess
		client *api.Client
	}

	tests := []struct {
		name               string
		args               args
		hr                 structs.HeightRange
		returnBlocks       map[uint64]structs.Block
		returnBlockMetaErr error
		expectMsgCount     int
	}{
		{name: "test with single height and zero transactions",
			hr: structs.HeightRange{
				StartHeight: 9870,
				EndHeight:   9871,
				ChainID:     "columbus-3",
				Network:     "terra",
			},
			args: args{
				ctx:    context.Background(),
				stream: cStructs.NewStreamAccess(),
			},
			returnBlocks: map[uint64]structs.Block{
				9870: structs.Block{Height: 9870, ChainID: "kava", NumberOfTransactions: 0},
			},
			expectMsgCount: 2,
		},
		{name: "test with single height and one transaction",
			hr: structs.HeightRange{
				StartHeight: 9870,
				EndHeight:   9871,
				ChainID:     "columbus-3",
				Network:     "terra",
			},
			args: args{
				ctx:    context.Background(),
				stream: cStructs.NewStreamAccess(),
			},
			returnBlocks: map[uint64]structs.Block{
				9870: structs.Block{Height: 9870, ChainID: "kava", NumberOfTransactions: 1},
			},
			expectMsgCount: 3,
		},
		{name: "test with single height and multiple transactions",
			hr: structs.HeightRange{
				StartHeight: 9870,
				EndHeight:   9871,
				ChainID:     "columbus-3",
				Network:     "terra",
			},
			args: args{
				ctx:    context.Background(),
				stream: cStructs.NewStreamAccess(),
			},
			returnBlocks: map[uint64]structs.Block{
				9870: structs.Block{Height: 9870, ChainID: "kava", NumberOfTransactions: 5},
			},
			expectMsgCount: 7,
		},
		{name: "test with multiple heights",
			hr: structs.HeightRange{
				StartHeight: 9870,
				EndHeight:   9873,
				ChainID:     "columbus-3",
				Network:     "terra",
			},
			args: args{
				ctx:    context.Background(),
				stream: cStructs.NewStreamAccess(),
			},
			returnBlocks: map[uint64]structs.Block{
				9870: structs.Block{Height: 9870, ChainID: "kava", NumberOfTransactions: 4},
				9871: structs.Block{Height: 9871, ChainID: "kava", NumberOfTransactions: 1},
				9872: structs.Block{Height: 9872, ChainID: "kava", NumberOfTransactions: 5},
			},
			expectMsgCount: 14,
		},
		{name: "test with client error",
			hr: structs.HeightRange{
				StartHeight: 9870,
				EndHeight:   9871,
				ChainID:     "columbus-3",
				Network:     "terra",
			},
			args: args{
				ctx:    context.Background(),
				stream: cStructs.NewStreamAccess(),
			},
			returnBlockMetaErr: fmt.Errorf("test err"),
			expectMsgCount:     1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockRPCCtrl := gomock.NewController(t)
			rpcMock := apiMocks.NewMockRPC(mockRPCCtrl)
			defer mockRPCCtrl.Finish()

			rpcMock.EXPECT().GetBlocksMeta(gomock.Any(), tt.hr, gomock.Any(), gomock.Any(), gomock.Any()).
				Do(
					func(ctx context.Context, params structs.HeightRange, limit uint64, blocks *api.BlocksMap, end chan<- error) {
						if tt.returnBlockMetaErr != nil {
							end <- tt.returnBlockMetaErr
							return
						}

						for i := tt.hr.StartHeight; i < tt.hr.EndHeight; i++ {
							currHeight := i
							blocks.Blocks[currHeight] = tt.returnBlocks[currHeight]
							end <- nil
						}
					},
				).Times(1)

			if tt.returnBlockMetaErr == nil {
				rpcMock.EXPECT().SingularHeightTxWorker(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
					func(ctx context.Context, wg *sync.WaitGroup, out chan types.TxResponse, in chan api.ToGet) {

						for req := range in {
							var i uint64
							for i = 0; i < tt.returnBlocks[req.Height].NumberOfTransactions; i++ {
								out <- types.TxResponse{Height: strconv.FormatUint(req.Height, 10)}
							}
						}
						wg.Done()
					},
				).Times(5)

				rpcMock.EXPECT().RawToTransactionCh(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(
					func(wg *sync.WaitGroup, in <-chan types.TxResponse, blocks map[uint64]structs.Block, out chan cStructs.OutResp) {
						for txRaw := range in {
							hInt, err := strconv.ParseUint(txRaw.Height, 10, 64)
							if err != nil {
								t.Errorf("unexpected error while parsing height: %v", err)
								return
							}
							out <- cStructs.OutResp{Type: "Transaction", Payload: structs.Transaction{Height: hInt}}
						}
						wg.Done()
					},
				)
			}

			payload, _ := json.Marshal(tt.hr)
			ic := NewIndexerClient(ctx, zaptest.NewLogger(t), nil, rpcMock, bigPage, maximumHeightsToGet)
			ic.GetTransactions(ctx, cStructs.TaskRequest{Payload: payload}, tt.args.stream, rpcMock)

			if len(tt.args.stream.ResponseListener) != tt.expectMsgCount {
				t.Errorf("unexpected stream.ResponseListener length. want %v, got %v", tt.expectMsgCount, len(tt.args.stream.ResponseListener))
				return
			}

			if tt.returnBlockMetaErr != nil {
				resp := <-tt.args.stream.ResponseListener
				if !strings.Contains(resp.Error.Msg, tt.returnBlockMetaErr.Error()) {
					t.Errorf("expected stream response to contain error msg: %v; got %v", tt.returnBlockMetaErr.Error(), resp.Error.Msg)
				}
				if resp.Final != true {
					t.Errorf("expected response to be final; got %v", resp)
				}
				return
			}

			// check that stream has 1 block and correct number of transaction for each height
			type counts struct {
				txs   uint64
				block uint64
			}
			respMap := make(map[uint64]*counts)
			for i := 0; i < tt.expectMsgCount; i++ {
				resp := <-tt.args.stream.ResponseListener

				switch resp.Type {
				case "Transaction":
					var data structs.Transaction
					err := json.Unmarshal(resp.Payload, &data)
					if err != nil {
						t.Errorf("unexpected error while unmarshalling payload: %v", err)
						return
					}
					c, ok := respMap[data.Height]
					if !ok {
						c = &counts{block: 0, txs: 0}
						respMap[data.Height] = c
					}
					c.txs++

				case "Block":
					var data structs.Block
					err := json.Unmarshal(resp.Payload, &data)
					if err != nil {
						t.Errorf("unexpected error while unmarshalling payload: %v", err)
						return
					}
					c, ok := respMap[data.Height]
					if !ok {
						c = &counts{block: 0, txs: 0}
						respMap[data.Height] = c
					}
					c.block++

				case "END":
					if len(tt.args.stream.ResponseListener) > 0 {
						t.Errorf("unexpected END message. Remaining messages in chan: %v", len(tt.args.stream.ResponseListener))
						return
					}
					if resp.Final != true {
						t.Errorf("expected response to be final; got %v", resp)
					}
				default:
					t.Errorf("unexpected message %v", resp.Type)
					return
				}
			}

			for i := tt.hr.StartHeight; i < tt.hr.EndHeight; i++ {
				c, ok := respMap[i]
				if !ok {
					t.Errorf("missing entry for height %v", i)
				}
				if c.block != 1 {
					t.Errorf("unexpected block count for height %v. want %v, got: %v", i, 1, c.block)
				}
				if c.txs != tt.returnBlocks[i].NumberOfTransactions {
					t.Errorf("unexpected tx count for height %v. want %v, got: %v", i, tt.returnBlocks[i].NumberOfTransactions, c.txs)
				}
			}
		})
	}
}

func TestIndexerClient_GetReward(t *testing.T) {
	var bigPage uint64 = 30
	var maximumHeightsToGet uint64 = 30

	type cliResp struct {
		data structs.GetRewardResponse
		err  error
	}
	tests := []struct {
		name    string
		ha      structs.HeightAccount
		cliResp cliResp
	}{
		{name: "Returns client response",
			ha: structs.HeightAccount{Height: 99, Account: "terra123"},
			cliResp: cliResp{structs.GetRewardResponse{
				Height:  99,
				Rewards: []structs.TransactionAmount{{Text: "100uluna"}, {Text: "200uluna"}},
			}, nil},
		},
		{name: "Returns error when client errors",
			ha:      structs.HeightAccount{Height: 99, Account: "terra123"},
			cliResp: cliResp{structs.GetRewardResponse{}, fmt.Errorf("oh noes")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCtrl := gomock.NewController(t)
			lcdMock := apiMocks.NewMockLCD(mockCtrl)
			defer mockCtrl.Finish()

			lcdMock.EXPECT().GetReward(gomock.Any(), tt.ha).Return(tt.cliResp.data, tt.cliResp.err).Times(1)
			ic := NewIndexerClient(ctx, zaptest.NewLogger(t), lcdMock, nil, bigPage, maximumHeightsToGet)

			payload, _ := json.Marshal(tt.ha)
			tr := cStructs.TaskRequest{
				Payload: payload,
			}

			stream := cStructs.NewStreamAccess()
			defer stream.Close()
			ic.GetReward(ctx, tr, stream, lcdMock)

			if len(stream.ResponseListener) == 0 {
				t.Errorf("expected stream.ResponseListener to not be empty")
			}

			resp1 := <-stream.ResponseListener
			if tt.cliResp.err != nil {
				if !strings.Contains(resp1.Error.Msg, tt.cliResp.err.Error()) {
					t.Errorf("expected stream response to contain error msg: %v; got %v", tt.cliResp.err.Error(), resp1.Error.Msg)
				}
				if resp1.Type != "END" && resp1.Final != true {
					t.Errorf("expected response to be final; got %v", resp1)
				}
				return
			}

			// if there's no error, expect first response to contain rewards data and not be final
			if resp1.Type != "Reward" && resp1.Final != false && resp1.Error.Msg != "" {
				t.Errorf("uexpected response; got %v", resp1)
			}

			var data structs.GetRewardResponse
			err := json.Unmarshal(resp1.Payload, &data)
			if err != nil {
				t.Errorf("unexpected error while unmarshalling payload: %v", err)
			}

			if data.Height != tt.cliResp.data.Height || len(data.Rewards) != len(tt.cliResp.data.Rewards) {
				t.Errorf("expected payload to contain data from client. want %v, got %v", tt.cliResp.data, data)
			}

			for _, got := range data.Rewards {
				var found bool
				for _, expect := range tt.cliResp.data.Rewards {
					if expect == got {
						found = true
						continue
					}
				}

				if !found {
					t.Errorf("unexpected rewards entry in payload. got %v", got)
				}

			}

			// if there's no error, expect a second response to be final
			if len(stream.ResponseListener) == 0 {
				t.Errorf("expected stream.ResponseListener to contain a final entry")
			}
			resp2 := <-stream.ResponseListener
			if resp2.Type != "END" && resp2.Final != true {
				t.Errorf("expected response to be final. got %v", resp2)
			}
		})
	}
}
