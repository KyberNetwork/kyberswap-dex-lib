package angstrom

import (
	"context"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/singleflight"
)

type RFQHandler struct {
	cfg *RFQConfig

	httpClient *resty.Client

	latestAttestations     []Attenstation
	latestAttestationsTime time.Time

	g singleflight.Group
}

func NewRFQHandler(cfg *RFQConfig) *RFQHandler {
	httpClient := resty.New().
		SetBaseURL(cfg.HTTP.BaseURL).
		SetTimeout(cfg.HTTP.Timeout.Duration).
		SetRetryCount(cfg.HTTP.RetryCount).
		SetHeader("X-API-KEY", cfg.HTTP.APIKey)

	return &RFQHandler{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

func (h *RFQHandler) RFQ(ctx context.Context, params pool.RFQParams) (*pool.RFQResult, error) {
	attestations, err := h.getAttestations(ctx)
	if err != nil {
		return nil, err
	}

	return &pool.RFQResult{
		Extra: &RFQExtra{
			Adapter:      Adapter,
			Attestations: attestations,
		},
	}, nil
}

func (h *RFQHandler) getAttestations(ctx context.Context) ([]Attenstation, error) {
	if time.Since(h.latestAttestationsTime) < h.cfg.CacheTTL.Duration {
		return h.latestAttestations, nil
	}

	_, err, _ := h.g.Do("rfqAttestations", func() (interface{}, error) {
		var resp AttenstationsResponse

		_, err := h.httpClient.R().
			SetBody(map[string]any{
				"blocks_in_future": h.cfg.BlocksInFuture,
			}).
			SetResult(&resp).
			Post(GET_ATTESTATIONS_PATH)

		if err != nil {
			return nil, err
		}

		h.latestAttestations = resp.Attestations
		h.latestAttestationsTime = time.Now()

		return h.latestAttestations, nil
	})

	if err != nil {
		return nil, err
	}

	return h.latestAttestations, nil
}
