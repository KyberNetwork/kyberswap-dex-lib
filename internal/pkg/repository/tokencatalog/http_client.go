package tokencatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/service"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

const (
	upsertEndpoint       = "/api/v1/internal/tokens"
	ContentTypeHeaderKey = "Content-Type"
)

type (
	httpClient struct {
		client *resty.Client
	}

	upsertResult struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	upsertRequest struct {
		ServiceCode string `json:"serviceCode"`
		Address     string `json:"address"`
		ChainID     string `json:"chainId"`
	}
)

func NewHTTPClient(baseURL string) service.ITokenCatalogRepository {
	client := resty.New()
	client.SetBaseURL(baseURL)
	return &httpClient{
		client,
	}
}

func (c *httpClient) Upsert(ctx context.Context, token service.CatalogToken) error {
	req := c.buildUpsertRequest(token)
	var result upsertResult
	resp, err := req.SetResult(&result).Post(upsertEndpoint)

	if err != nil {
		return err
	}
	if result.Code != 0 {
		return errors.New(result.Message)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 400 {
		return fmt.Errorf("error when performing Upsert with response=%v", resp)
	}
	logger.WithFields(logger.Fields{
		"token": token,
	}).Infof("Upsert successful to token catalog with token")
	return nil
}

func (c *httpClient) buildUpsertRequest(token service.CatalogToken) *resty.Request {
	req := c.client.R().
		SetHeader(ContentTypeHeaderKey, "application/json").
		SetBody(upsertRequest{
			ServiceCode: token.Source,
			Address:     token.Address,
			ChainID:     token.ChainID,
		})
	return req
}
