package client

import (
	"context"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	"github.com/stretchr/testify/assert"
)

func TestHTTPClient(t *testing.T) {
	t.Skip("has rate-limit for non-authorization requests")

	c := NewHTTPClient(
		&dexalot.HTTPClientConfig{
			BaseURL: "https://api.dexalot.com",
			Timeout: durationjson.Duration{
				Duration: time.Second * 5,
			},
			RetryCount: 1,
			APIKey:     "",
		},
	)

	_, err := c.Quote(context.Background(), dexalot.FirmQuoteParams{
		ChainID:     43114,
		TakerAsset:  "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
		MakerAsset:  "0x0000000000000000000000000000000000000000",
		TakerAmount: "200000000",
		UserAddress: "0x05A1AAC00662ADda4Aa25E1FA658f4256ed881eD",
		Executor:    "0xdef171fe48cf0115b1d80b88dc8eab59176fee57",
	}, 20)
	assert.NoError(t, err)
}
