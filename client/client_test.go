package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/figment-networks/indexer-manager/structs"
	cStructs "github.com/figment-networks/indexer-manager/worker/connectivity/structs"
	"github.com/figment-networks/terra-worker/api"
	apiMocks "github.com/figment-networks/terra-worker/client/mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestIndexerClient_GetTransactions(t *testing.T) {
	type fields struct {
		cli                 *api.Client
		logger              *zap.Logger
		streams             map[uuid.UUID]*cStructs.StreamAccess
		sLock               sync.Mutex
		bigPage             uint64
		maximumHeightsToGet uint64
	}
	type args struct {
		ctx    context.Context
		tr     cStructs.TaskRequest
		stream *cStructs.StreamAccess
		client *api.Client
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockRPCCtrl := gomock.NewController(t)
			rpcMock := apiMocks.NewMockRPC(mockRPCCtrl)
			defer mockRPCCtrl.Finish()

			mockLCDCtrl := gomock.NewController(t)
			lcdMock := apiMocks.NewMockLCD(mockLCDCtrl)
			defer mockLCDCtrl.Finish()

			ic := NewIndexerClient(ctx, zaptest.NewLogger(t), lcdMock, rpcMock, tt.fields.bigPage, tt.fields.maximumHeightsToGet)
			ic.GetTransactions(ctx, tt.args.tr, tt.args.stream, tt.args.client)

			/*			ic := &IndexerClient{
							cli:                 tt.fields.cli,
							logger:              tt.fields.logger,
							streams:             tt.fields.streams,
							sLock:               tt.fields.sLock,
							bigPage:             tt.fields.bigPage,
							maximumHeightsToGet: tt.fields.maximumHeightsToGet,
						}
			*/
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
