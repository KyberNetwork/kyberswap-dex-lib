package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bebop/client"
	"github.com/stretchr/testify/assert"
)

func TestHTTPClient(t *testing.T) {
	t.Skip("has rate-limit for non-authorization requests")

	c := client.NewHTTPClient(
		&bebop.HTTPClientConfig{
			BaseURL: "https://api.bebop.xyz/pmm/ethereum",
			Timeout: durationjson.Duration{
				Duration: time.Second * 5,
			},
			RetryCount:    1,
			Name:          "",
			Authorization: "",
		},
	)

	resp, err := c.QuoteSingleOrderResult(context.Background(), bebop.QuoteParams{
		SellTokens:   "0xC02aaA39b223fe8D0A0e5C4F27eAD9083C756Cc2",
		BuyTokens:    "0xdac17F958D2ee523a2206206994597C13D831ec7",
		SellAmounts:  "100000000000000000",
		TakerAddress: "0x5Bad996643a924De21b6b2875c85C33F3c5bBcB6",
		ApprovalType: "Standard",
	})
	assert.NoError(t, err)

	t.Log(resp.ToSign)
}
