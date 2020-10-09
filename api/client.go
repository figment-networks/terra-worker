package api

import (
	"net/http"
	"time"

	amino "github.com/tendermint/go-amino"
	"github.com/terra-project/core/app"
	"github.com/terra-project/core/x/auth"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingCosmos "github.com/cosmos/cosmos-sdk/x/slashing"




	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var cdcA = amino.NewCodec()

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL     string
	key         string
	httpClient  *http.Client
	logger      *zap.Logger
	rateLimiter *rate.Limiter
	cdc         *amino.Codec
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, logger *zap.Logger, c *http.Client, reqPerSecLimit int) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}
	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	cdc := app.MakeCodec()
	sdk.RegisterCodec(cdcA)
	slashingCosmos.RegisterCodec(cdcA)
	auth.RegisterCodec(cdcA)

	cli := &Client{
		logger:      logger,
		baseURL:     url, //tendermint rpc url
		key:         key,
		httpClient:  c,
		rateLimiter: rateLimiter,
		cdc:         cdc,
	}

	return cli
}

func (c *Client) CDC() *amino.Codec {
	return c.cdc
}

// InitMetrics initialise metrics
func InitMetrics() {
	convertionDurationObserver = conversionDuration.WithLabels("conversion")
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	blockCacheEfficiencyHit = blockCacheEfficiency.WithLabels("hit")
	blockCacheEfficiencyMissed = blockCacheEfficiency.WithLabels("missed")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
}
