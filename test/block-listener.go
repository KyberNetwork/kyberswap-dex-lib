package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// client, err := ethclient.Dial("wss://speedy-nodes-nyc.moralis.io/31e7e21cccc77b252959d97b/polygon/mainnet/ws")
	client, err := ethclient.Dial("wss://polygon-ws.knstats.com/v1/mainnet/geth?appId=dev-dmm-aggregator-backend")
	if err != nil {
		log.Fatal(err)
	}

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Block number: %d, hash: %s, timestamp: %+v\n", block.Number().Uint64(), header.Hash().Hex(), block.Time())
		}
	}
}
