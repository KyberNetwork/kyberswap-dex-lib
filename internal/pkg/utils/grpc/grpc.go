package grpc

import (
	"github.com/KyberNetwork/service-framework/pkg/client/grpcclient"

	"google.golang.org/grpc"
)

func NewClient[T any](clientFactory func(grpc.ClientConnInterface) T, config grpcclient.Config) (*grpcclient.Client[T], error) {
	client, err := grpcclient.New(clientFactory, grpcclient.WithConfig(&config))
	if err != nil {
		return nil, err
	}

	return client, nil
}
