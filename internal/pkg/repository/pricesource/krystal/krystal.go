package krystal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/backoff"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type krystalPriceSource struct {
	baseURL string
	client  *http.Client
}

func NewKrystalPriceSource(
	baseURL string,
	timeout time.Duration,
) repository.IPriceSourceRepository {
	return &krystalPriceSource{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

type PriceResponse struct {
	MarketData []TokenPriceItem `json:"marketData"`
}

type TokenPriceItem struct {
	Address string  `json:"address"`
	Price   float64 `json:"price"`
}

func (s *krystalPriceSource) DoGet(ctx context.Context, url string, dest interface{}) error {
	resp, err := s.client.Get(url)
	if err != nil {
		logger.Errorf("failed to call GET api, err: %v", err)
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("url not found")
		logger.Errorf("failed to call GET api, err: %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request is not 200, status code: %d, status: %v", resp.StatusCode, resp.Status)
		logger.Errorf("failed to call GET api, err: %v", err)
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed to parse GET response, err: %v", err)
		return err
	}

	err = json.Unmarshal(body, dest)
	if err != nil {
		logger.Errorf("failed to unmarshal response data, err: %v", err)
		return err
	}

	return nil
}

func (s *krystalPriceSource) FetchPrice(ctx context.Context, cfg *config.Common, tokens []string) (map[string]float64, error) {
	url := fmt.Sprintf(
		"%s/market?addresses=%s&chain=ethereum@%d&sparkline=false",
		s.baseURL,
		strings.Join(tokens, ","),
		cfg.ChainID,
	)

	var response PriceResponse

	var err error
	backoff.Retry(
		func() error {
			if err = s.DoGet(ctx, url, &response); err == nil || strings.Contains(err.Error(), "not found") {
				return nil
			}

			logger.Errorf("failed to call Krystal api, url: %v, err: %v", url, err)

			return err
		},
	)

	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)

	for _, tokenPriceItem := range response.MarketData {
		result[tokenPriceItem.Address] = tokenPriceItem.Price
	}

	return result, nil
}
