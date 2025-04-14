package ekubo

import (
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
)

func clientFromEnv() (*ethclient.Client, error) {
	return ethclient.Dial(os.Getenv("ETH_RPC_URL"))
}
