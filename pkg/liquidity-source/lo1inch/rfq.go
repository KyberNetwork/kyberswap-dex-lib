package lo1inch

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type RFQHandler struct {
	pool.RFQHandler
	config *Config
}

func NewRFQHandler(config *Config) *RFQHandler {
	return &RFQHandler{
		config: config,
	}
}

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	return nil, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []pool.RFQParams) ([]*pool.RFQResult, error) {
	return nil, nil
}
