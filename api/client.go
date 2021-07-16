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

func init() {
	sdk.RegisterCodec(cdcA)
	slashingCosmos.RegisterCodec(cdcA)
	auth.RegisterCodec(cdcA)
}

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL     string
	key         string
	chainID     string
	httpClient  *http.Client
	logger      *zap.Logger
	rateLimiter *rate.Limiter
	cdc         *amino.Codec
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key, chainID string, logger *zap.Logger, c *http.Client, reqPerSecLimit int) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}
	rateLimiter := rate.NewLimiter(rate.Limit(reqPerSecLimit), reqPerSecLimit)

	cdc := app.MakeCodec()

	cli := &Client{
		logger:      logger,
		baseURL:     url, //tendermint rpc or terra lcd url
		key:         key,
		httpClient:  c,
		chainID:     chainID,
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
	transactionConversionDuration = conversionDuration.WithLabels("transaction")
	numberOfItemsTransactions = numberOfItems.WithLabels("transactions")
	numberOfItemsInBlock = numberOfItemsBlock.WithLabels("transactions")
}
