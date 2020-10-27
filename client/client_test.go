package client

import (
	"context"
	"sync"
	"testing"

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

			mockApiCtrl := gomock.NewController(t)
			apiMock := apiMocks.NewMockApi(mockApiCtrl)
			defer mockApiCtrl.Finish()

			ic := NewIndexerClient(ctx, zaptest.NewLogger(t), apiMock, tt.fields.bigPage, tt.fields.maximumHeightsToGet)
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
