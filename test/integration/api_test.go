package integration

/*
import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/figment-networks/indexing-engine/structs"
	"github.com/figment-networks/terra-worker/api"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap/zaptest"
)

func TestGetAccountBalance(t *testing.T) {
	lcdAddr := "https://columbus-4--lcd--archive.datahub.figment.io"
	dataHubKey := "" // set your api key before testing
	tests := []struct {
		name       string
		lcdAddr    string
		dataHubKey string
		args       structs.HeightAccount
		res        map[string]string
		wantErr    bool
	}{
		{
			name:       "wrong account address syntax",
			lcdAddr:    lcdAddr,
			dataHubKey: dataHubKey,
			args: structs.HeightAccount{
				Account: "wrong account address",
			},
			wantErr: true,
		},
		{
			name:       "present account address",
			lcdAddr:    lcdAddr,
			dataHubKey: dataHubKey,
			args: structs.HeightAccount{
				Account: "terra15cupwhpnxhgylxa8n4ufyvux05xu864jcrrkqa",
				Height:  10000,
			},
			res: map[string]string{
				"ukrw":  "3969048309959",
				"uluna": "154091201348",
				"umnt":  "16081798865",
				"usdr":  "6931297",
				"uusd":  "671011",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			zl := zaptest.NewLogger(t)
			capi := api.NewClient(tt.lcdAddr, tt.dataHubKey, zl, nil, 10)
			resp, err := capi.GetAccountBalance(ctx, tt.args)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if len(resp.Balances) != len(tt.res) {
					require.NoError(t, errors.New("unexpected"))
				}
				var n *big.Int

				for _, blnc := range resp.Balances {
					value, ok := tt.res[blnc.Currency]
					if !ok {
						require.NoError(t, errors.New("unexpected"))
					}
					require.Equal(t, blnc.Text, value)
					n = new(big.Int)
					n.SetString(value, 10)
					require.Equal(t, blnc.Numeric, n)
					require.Equal(t, blnc.Exp, int32(0)) // not available for terra
				}

			}
		})
	}
}
*/
