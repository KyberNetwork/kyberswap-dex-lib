package router

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pooldecoder "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/decode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

const (
	getOrderPlaceholderPath string = "/{chain}/order/{orderHash}"
)

type Config struct {
	Router     string            `json:"router,omitempty"`
	Chain      string            `json:"chain,omitempty"`
	HTTPClient *HTTPClientConfig `mapstructure:"http_client" json:"httpClient"`
}

type HTTPClientConfig struct {
	BaseURL          string        `mapstructure:"base_url" json:"baseUrl"`
	Timeout          time.Duration `mapstructure:"timeout" json:"timeout"`
	RetryCount       int           `mapstructure:"retry_count" json:"retryCount"`
	AuthorizationKey string        `mapstructure:"authorization_key" json:"authorizationKey"`
}

type EventParser struct {
	config *Config
	client I1inchClient
}

type OrderResp struct {
	makerAsset  string `json:"makerAsset"`
	takerAsset  string `json:"takerAsset"`
	PoolAddress string `json:"poolAddress"`
}

type Client1inch struct {
	*resty.Client
}

func New1inchClient(client *resty.Client) *Client1inch {
	return &Client1inch{
		Client: client,
	}
}

func (c *Client1inch) GetOrder(ctx context.Context, getOrderPath string) (*OrderResp, error) {
	order, err := c.R().SetContext(ctx).Get(getOrderPath)
	if err != nil {
		return nil, err
	}
	// TODO: order not found handle
	var orderResp OrderResp
	if err := json.Unmarshal(order.Body(), &orderResp); err != nil {
		return nil, err
	}
	return &orderResp, nil
}

var _ = pooldecoder.RegisterFactoryC(Type, NewEventParser)

func NewEventParser(config *Config) *EventParser {
	client := resty.New()
	client.SetBaseURL(config.HTTPClient.BaseURL).
		SetTimeout(config.HTTPClient.Timeout).
		SetRetryCount(config.HTTPClient.RetryCount).
		SetHeader("Authorization", "Bearer "+config.HTTPClient.AuthorizationKey).
		SetHeader("Content-Type", "application/json")
	return &EventParser{
		config: config,
		client: New1inchClient(client),
	}
}

func (p *EventParser) SetClient(client I1inchClient) {
	p.client = client
}

func (p *EventParser) Decode(ctx context.Context, logs []types.Log) (map[string][]types.Log, error) {
	routerAddress, err := p.GetKey(ctx)
	if err != nil {
		return nil, err
	}
	addressLogs := make(map[string][]types.Log)
	for _, log := range logs {
		if log.Address != common.HexToAddress(routerAddress) {
			continue
		}
		switch log.Topics[0] {
		case RouterABI.Events["OrderFilled"].ID:
			orderHash := common.BytesToHash(log.Data[:32]).Hex()
			getOrderPath := strings.ReplaceAll(getOrderPlaceholderPath, "{chain}", p.config.Chain)
			getOrderPath = strings.ReplaceAll(getOrderPath, "{orderHash}", orderHash)
			orderResp, err := p.client.GetOrder(ctx, getOrderPath)
			if err != nil {
				return nil, err
			}
			makerAsset := strings.ToLower(orderResp.makerAsset)
			takerAsset := strings.ToLower(orderResp.takerAsset)

			orderResp.PoolAddress = lo.Ternary(makerAsset < takerAsset,
				util.FormatKey("_", "lo1inch", makerAsset, takerAsset),
				util.FormatKey("_", "lo1inch", takerAsset, makerAsset),
			)
			addressLogs[orderResp.PoolAddress] = append(addressLogs[orderResp.PoolAddress], log)
		}
	}
	return addressLogs, nil
}

func (p *EventParser) GetKey(ctx context.Context) (poolAddress string, err error) {
	if p.config.Router == "" {
		return "", errors.New("lo1inch router address is not set")
	}
	return strings.ToLower(p.config.Router), nil
}
