package eth

import (
	"errors"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ErrNoConfiguredPRC = errors.New("no configured rpc")
)

func NewClient(rpcs []string) (*ethclient.Client, error) {
	if len(rpcs) == 0 {
		return nil, ErrNoConfiguredPRC
	}

	rand.Seed(time.Now().UnixNano())

	for {
		randIndex := rand.Intn(len(rpcs))
		rpc := rpcs[randIndex]

		client, err := ethclient.Dial(rpc)
		if err != nil {
			rpcs = append(rpcs[:randIndex], rpcs[randIndex+1:]...)

			continue
		}

		return client, nil
	}
}
