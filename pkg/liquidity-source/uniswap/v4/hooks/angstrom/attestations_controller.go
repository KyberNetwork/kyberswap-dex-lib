package angstrom

import (
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/go-resty/resty/v2"
)

type AttestationController struct {
	httpClient *resty.Client
	cfg        HookConfig

	latestAttestations     []Attestation
	latestAttestationsTime time.Time
}

var (
	once     sync.Once
	instance *AttestationController
)

func GetAttestationController(config HookConfig) *AttestationController {
	once.Do(func() {
		httpClient := resty.New().
			SetBaseURL(config.HTTP.BaseURL).
			SetTimeout(config.HTTP.Timeout.Duration).
			SetRetryCount(config.HTTP.RetryCount).
			SetHeader("X-API-KEY", config.HTTP.APIKey)

		instance = &AttestationController{
			cfg:        config,
			httpClient: httpClient,
		}

		go func() {
			ticker := time.NewTicker(config.RefreshInterval.Duration)
			defer ticker.Stop()

			for {
				logger.Debug("fetching latest attestations from angstrom hook")
				_, err := instance.fetchAttestations()
				if err != nil {
					logger.WithFields(logger.Fields{
						"error": err,
					}).Error("failed to fetch attestations periodically")
				}

				<-ticker.C
			}
		}()
	})

	return instance
}

func (h *AttestationController) GetLatestAttestations() ([]Attestation, time.Time) {
	return h.latestAttestations, h.latestAttestationsTime
}

func (h *AttestationController) fetchAttestations() ([]Attestation, error) {
	var resp AttestationsResponse

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
}
