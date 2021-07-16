package api

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"google.golang.org/grpc"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type ClientConfig struct {
	ReqPerSecond        int
	TimeoutBlockCall    time.Duration
	TimeoutSearchTxCall time.Duration
}

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	chainID string
	logger  *zap.Logger
	Sbc     *SimpleBlockCache

	// GRPC
	tmServiceClient tmservice.ServiceClient
	txServiceClient tx.ServiceClient
	rateLimiterGRPC *rate.Limiter

	cfg *ClientConfig
}

// NewClient returns a new client for a given endpoint
func NewClient(chainID string, logger *zap.Logger, cli *grpc.ClientConn, cfg *ClientConfig) *Client {
	rateLimiterGRPC := rate.NewLimiter(rate.Limit(cfg.ReqPerSecond), cfg.ReqPerSecond)

	return &Client{
		chainID: chainID,
		logger:  logger,
		Sbc:     NewSimpleBlockCache(400),

		rateLimiterGRPC: rateLimiterGRPC,
		tmServiceClient: tmservice.NewServiceClient(cli),
		txServiceClient: tx.NewServiceClient(cli),

		cfg: cfg,
	}
}

// InitMetrics initialise metrics
func InitMetrics() {
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
	numberOfItemsInBlock = numberOfItemsBlock.WithLabels("transactions")
}
