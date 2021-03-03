package api

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"strconv"

	"github.com/figment-networks/indexer-manager/structs"
	"net"
	"net/http"
	"time"

	"encoding/json"
)

type responseBalance struct {
	Height string     `json:"height"`
	Result []sdk.Coin `json:"result"`
}

// GetAccountBalance fetches account balance
func (c *Client) GetAccountBalance(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountBalanceResponse, err error) {
	resp.Height = params.Height
	endpoint := fmt.Sprintf("%s/bank/balances/%v", c.baseURL, params.Account)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return resp, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", c.key)

	q := req.URL.Query()
	if params.Height > 0 {
		q.Add("height", strconv.FormatUint(params.Height, 10))
	}

	req.URL.RawQuery = q.Encode()

	err = c.rateLimiter.Wait(ctx)
	if err != nil {
		return resp, err
	}

	var cliResp *http.Response

	for i := 1; i <= maxRetries; i++ {
		n := time.Now()
		cliResp, err = c.httpClient.Do(req)
		if err, ok := err.(net.Error); ok && err.Timeout() && i != maxRetries {
			continue
		} else if err != nil {
			return resp, err
		}
		rawRequestDuration.WithLabels(endpoint, cliResp.Status).Observe(time.Since(n).Seconds())

		defer cliResp.Body.Close()

		if cliResp.StatusCode < 500 {
			break
		}
		time.Sleep(time.Duration(i*500) * time.Millisecond)
	}

	decoder := json.NewDecoder(cliResp.Body)

	if cliResp.StatusCode > 399 {
		var result rest.ErrorResponse
		if err = decoder.Decode(&result); err != nil {
			return resp, fmt.Errorf("[TERRA-API] Error fetching account balance: %d", cliResp.StatusCode)
		}
		return resp, fmt.Errorf("[TERRA-API] Error fetching account balance: %s ", result.Error)
	}
	var result responseBalance
	if err = decoder.Decode(&result); err != nil {
		return resp, err
	}

	for _, blnc := range result.Result {
		resp.Balances = append(resp.Balances,
			structs.TransactionAmount{
				Text:     blnc.Amount.String(),
				Numeric:  blnc.Amount.BigInt(),
				Currency: blnc.Denom,
			},
		)
	}

	return resp, err
}
