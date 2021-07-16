package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api/types"
)

var (
	tenInt = big.NewInt(10)

	// stakingCurrency is always uluna, https://medium.com/figment/terra-staking-delegation-guide-for-luna-tokens-32383f2f959f
	stakingCurrency = "uluna"
)

// GetAccountDelegations fetches account delegations
func (c *Client) GetAccountDelegations(ctx context.Context, params structs.HeightAccount) (resp structs.GetAccountDelegationsResponse, err error) {
	resp.Height = params.Height

	endpoint := fmt.Sprintf("/staking/delegators/%v/delegations", params.Account)
	req, err := http.NewRequest(http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return resp, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

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
		rawRequestHTTPDuration.WithLabels("/distribution/delegators/_/delegations", cliResp.Status).Observe(time.Since(n).Seconds())

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
			return resp, fmt.Errorf("[TERRA-API] Error fetching rewards: %d", cliResp.StatusCode)
		}
		return resp, fmt.Errorf("[TERRA-API] Error fetching rewards: %s ", result.Error)
	}

	if c.chainID == "columbus-4" {
		err = decodeDelegationsColumbus4(decoder, &resp)
	} else {
		err = decodeDelegationsColumbus3(decoder, &resp)

	}

	return resp, err
}

func decodeDelegationsColumbus4(decoder *json.Decoder, resp *structs.GetAccountDelegationsResponse) (err error) {
	var result types.DelegationResponseV4
	if err = decoder.Decode(&result); err != nil {
		return err
	}

	if len(result.Delegations) < 1 {
		return nil
	}

	for _, del := range result.Delegations {
		shareInt, shareExp, err := gettIntAndExp(del.Shares)
		if err != nil {
			return fmt.Errorf("could not convert shares, %w", err)
		}

		amtInt, amtExp, err := gettIntAndExp(del.Balance.Amount)
		if err != nil {
			return fmt.Errorf("could not convert shares, %w", err)
		}

		resp.Delegations = append(resp.Delegations,
			structs.Delegation{
				Delegator: del.DelegatorAddress,
				Validator: structs.Validator(del.ValidatorAddress),
				Shares: structs.RewardAmount{
					Numeric: shareInt,
					Exp:     shareExp,
				},
				Balance: structs.RewardAmount{
					Numeric:  amtInt,
					Currency: del.Balance.Denom,
					Exp:      amtExp,
				},
			},
		)
	}
	return
}

func decodeDelegationsColumbus3(decoder *json.Decoder, resp *structs.GetAccountDelegationsResponse) (err error) {
	var result types.DelegationResponse
	if err = decoder.Decode(&result); err != nil {
		return err
	}

	if len(result.Delegations) < 1 {
		return nil
	}

	for _, del := range result.Delegations {
		shareInt, shareExp, err := gettIntAndExp(del.Shares)
		if err != nil {
			return fmt.Errorf("could not convert shares, %w", err)
		}

		amtInt, amtExp, err := gettIntAndExp(del.Balance)
		if err != nil {
			return fmt.Errorf("could not convert shares, %w", err)
		}

		resp.Delegations = append(resp.Delegations,
			structs.Delegation{
				Delegator: del.DelegatorAddress,
				Validator: structs.Validator(del.ValidatorAddress),
				Shares: structs.RewardAmount{
					Numeric: shareInt,
					Exp:     shareExp,
				},
				Balance: structs.RewardAmount{
					Numeric:  amtInt,
					Currency: stakingCurrency,
					Exp:      amtExp,
				},
			},
		)
	}
	return
}

// gettIntAndExp converts a string of an integer or float  and converts it
// to a bigInt type, returning the int and number of decimal places
// so "44.55" becomes 4455, 2
func gettIntAndExp(s string) (*big.Int, int32, error) {
	if s == "" {
		return nil, 0, errors.New("foo")
	}

	parts := strings.Split(s, ".")
	intgrPart := parts[0]
	intgr, ok := new(big.Int).SetString(intgrPart, 10)
	if !ok {
		return nil, 0, errors.New("foo")
	}

	if len(parts) == 1 {
		return intgr, 0, nil
	} else if len(parts) > 2 {
		return nil, 0, errors.New("foo")
	}

	decPart := parts[1]
	dec, ok := new(big.Int).SetString(decPart, 10)
	if !ok {
		return nil, 0, errors.New("foo")
	}

	if dec.Cmp(big.NewInt(0)) == 0 {
		return intgr, 0, nil
	}

	exp := int32(len(decPart))
	mltplr := new(big.Int).Exp(tenInt, big.NewInt(int64(exp)), nil)
	intgr.Mul(intgr, mltplr)
	return intgr.Add(intgr, dec), exp, nil
}
