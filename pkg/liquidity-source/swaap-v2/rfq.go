package swaapv2

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/swaap-v2/client"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Config struct {
}

type IClient interface {
	Quote(ctx context.Context, params client.QuoteParams) (client.QuoteResult, error)
}

type RFQHandler struct {
	config *Config
	client IClient
}

func (h *RFQHandler) RFQ(ctx context.Context, recipient string, params any) (pool.RFQResult, error) {

	return pool.RFQResult{}, nil
}
