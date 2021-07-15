package api

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"google.golang.org/grpc"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// var cdcA = amino.NewCodec()

// func init() {
// 	sdk.RegisterCodec(cdcA)
// 	slashingCosmos.RegisterCodec(cdcA)
// 	auth.RegisterCodec(cdcA)
// }

type ClientConfig struct {
	ReqPerSecond        int
	TimeoutBlockCall    time.Duration
	TimeoutSearchTxCall time.Duration
}

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	Sbc *SimpleBlockCache

	baseURL string
	key     string
	chainID string
	// httpClient  *http.Client
	logger          *zap.Logger
	rateLimiter     *rate.Limiter
	rateLimiterGRPC *rate.Limiter

	// cdc         *amino.Codec

	// GRPC
	tmServiceClient tmservice.ServiceClient
	txServiceClient tx.ServiceClient

	cfg *ClientConfig
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key, chainID string, logger *zap.Logger, cli *grpc.ClientConn, reqPerSecLimit int, cfg *ClientConfig) *Client {
	// if c == nil {
	// 	c = &http.Client{
	// 		Timeout: time.Second * 40,
	// 	}
	// }
	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)
	rateLimiterGRPC := rate.NewLimiter(rate.Limit(cfg.ReqPerSecond), cfg.ReqPerSecond)

	fmt.Println("cli ==nil?", cli == nil)

	// cdc := app.MakeCodec()

	return &Client{
		logger:  logger,
		baseURL: url, //tendermint rpc or terra lcd url
		key:     key,
		Sbc:     NewSimpleBlockCache(400),

		// httpClient:  c,
		chainID:         chainID,
		rateLimiter:     rateLimiter,
		rateLimiterGRPC: rateLimiterGRPC,

		tmServiceClient: tmservice.NewServiceClient(cli),
		txServiceClient: tx.NewServiceClient(cli),

		// cdc:         cdc,
		cfg: cfg,
	}
}

// func (c *Client) CDC() *amino.Codec {
// 	return c.cdc
// }

// InitMetrics initialise metrics
func InitMetrics() {
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
	numberOfItemsInBlock = numberOfItemsBlock.WithLabels("transactions")
}
