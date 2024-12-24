package token

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/go-resty/resty/v2"
)

const (
	GetTokens = "/api/v1/internal/tokens"
)

type (
	httpClient struct {
		client *resty.Client
	}

	HttpConfig struct {
		Timeout time.Duration `mapstructure:"timeout"`
		BaseUrl string        `mapstructure:"baseUrl"`
	}

	listTokensResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Tokens []*routerEntity.TokenInfo `json:"tokens"`
		} `json:"data"`
	}
)

func NewHTTPClient(config HttpConfig) *httpClient {
	client := resty.New().SetTimeout(config.Timeout)
	client.SetBaseURL(config.BaseUrl).
		SetTimeout(config.Timeout).
		SetHeader("Content-Type", "application/json")
	return &httpClient{
		client: client,
	}
}

/*
 * Accepted maximum tokens is 100, no need to implement paging here
 */
func (c *httpClient) FindTokenInfos(ctx context.Context, chainID valueobject.ChainID, addresses []string) ([]*routerEntity.TokenInfo, error) {
	req := c.client.R().SetContext(ctx).
		SetQueryParams(map[string]string{
			"chainIds":  strconv.Itoa(int(chainID)),
			"addresses": strings.Join(addresses, ","),
		})

	var response listTokensResponse
	resp, err := req.SetResult(&response).Get(GetTokens)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("error ListTokens with response %v", resp)
	}

	if response.Code != 0 {
		return nil, errors.New(response.Message)
	}

	return response.Data.Tokens, nil
}
