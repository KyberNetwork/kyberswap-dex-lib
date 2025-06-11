package poolservice

import (
	"context"
	"strconv"
	"time"

	poolv1 "github.com/KyberNetwork/grpc-service/go/pool/v1"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/service-framework/pkg/client/grpcclient"
	"google.golang.org/grpc/metadata"
)

var chainHeader = "X-Chain-Id"

type GRPCPoolClient struct {
	client poolv1.PoolServiceClient
	config Config
}

func NewGRPCClient(config Config) (*GRPCPoolClient, error) {
	grpcConfig := grpcclient.Config{
		BaseURL:  config.BaseURL,
		Timeout:  config.Timeout,
		Insecure: config.Insecure,
		ClientID: config.ClientID,
	}

	client, err := grpcclient.New(poolv1.NewPoolServiceClient, grpcclient.WithConfig(&grpcConfig))
	if err != nil {
		return nil, err
	}

	return &GRPCPoolClient{
		client: client.C,
		config: config,
	}, nil
}

func (c *GRPCPoolClient) setHeaders(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, chainHeader, strconv.Itoa(int(c.config.ChainID)))
}

func (c *GRPCPoolClient) TrackFaultyPools(ctx context.Context, trackers []entity.FaultyPoolTracker) ([]string, error) {
	input := make([]*poolv1.FaultyPoolTracker, 0, len(trackers))
	for _, t := range trackers {
		input = append(input, &poolv1.FaultyPoolTracker{
			Address:     t.Address,
			FailedCount: t.FailedCount,
			TotalCount:  t.TotalCount,
		})
	}
	res, err := c.client.TrackFaultyPools(c.setHeaders(ctx), &poolv1.TrackFaultyPoolsRequest{
		Trackers: input,
	})

	if res != nil {
		return res.Addresses, err
	}

	return []string{}, err
}

func (c *GRPCPoolClient) GetFaultyPools(ctx context.Context, offset int64, count int64) ([]string, error) {
	res, err := c.client.GetFaultyPools(c.setHeaders(ctx), &poolv1.GetFaultyPoolsRequest{
		Offset: offset,
		Count:  count,
	})

	if err != nil {
		return []string{}, err
	}

	addresses := []string{}
	for _, p := range res.FaultyPools {
		addresses = append(addresses, p.Address)
	}

	return addresses, err
}

func (c *GRPCPoolClient) AddFaultyPools(ctx context.Context, faultyPools []entity.FaultyPool) error {
	pools := make([]*poolv1.FaultyPool, 0, len(faultyPools))
	for _, p := range faultyPools {
		pools = append(pools, &poolv1.FaultyPool{
			Address:   p.Address,
			ExpiredAt: p.ExpiresAt.Format(time.RFC3339),
		})
	}

	_, err := c.client.AddFaultyPools(c.setHeaders(ctx), &poolv1.AddFaultyPoolsRequest{
		Pools: pools,
	})

	return err
}
