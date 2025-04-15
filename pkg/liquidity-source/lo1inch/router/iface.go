package router

import "context"

//go:generate mockgen -destination ./client_1inch_mock.go -package router github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/router I1inchClient
type I1inchClient interface {
	GetOrder(ctx context.Context, getOrderPath string) (*OrderResp, error)
}
