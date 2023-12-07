package blackjack

import (
	"context"

	blackjackv1 "github.com/KyberNetwork/blackjack/proto/gen/blackjack/v1"
	"github.com/KyberNetwork/service-framework/pkg/client/grpcclient"
	"github.com/samber/lo"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/grpc"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/tracer"
)

type GRPCClient struct {
	client blackjackv1.ServiceClient
	config GRPCClientConfig
}

func NewGRPCClient(config GRPCClientConfig) (*GRPCClient, error) {
	grpcClientConfig := grpcclient.Config{
		BaseURL:  config.BaseURL,
		Timeout:  config.Timeout,
		Insecure: config.Insecure,
		ClientID: config.ClientID,
	}

	client, err := grpc.NewClient[blackjackv1.ServiceClient](blackjackv1.NewServiceClient, grpcClientConfig)
	if err != nil {
		return nil, err
	}

	return &GRPCClient{
		client: client.C,
		config: config,
	}, nil
}

func (c *GRPCClient) Check(ctx context.Context, wallets []string) (map[string]bool, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "[blackjack] GRPCClient.Check")
	defer span.End()

	async := true
	resp, err := c.client.Check(ctx, &blackjackv1.CheckRequest{
		Wallets: wallets,
		Async:   &async,
	})
	if err != nil {
		return nil, err
	}

	if resp.Data == nil || len(resp.Data.Wallets) == 0 {
		return nil, nil
	}

	result := lo.SliceToMap(resp.Data.Wallets, func(data *blackjackv1.BlacklistData) (string, bool) {
		return data.GetWallet(), data.GetBlacklisted()
	})

	return result, nil
}
