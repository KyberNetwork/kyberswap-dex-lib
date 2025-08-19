package main

import (
	"cmp"
	"context"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sync/atomic"

	dexentity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	erc20balanceslotuc "github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

const (
	HashKeyReadChunkSize = 10_000
)

func main() {
	app := &cli.App{
		Usage: "ERC20 balance slot",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "internal/pkg/config/default.yaml",
				Usage:   "Configuration file",
			},
			&cli.StringFlag{
				Name:    "safetyQuoting",
				Aliases: []string{"sq"},
				Value:   "internal/pkg/config/files/quoting-factor/tokens-v1.0.yaml",
				Usage:   "Configuration file",
			},
		},
		DefaultCommand: "probe-balance-slot",
		Commands: []*cli.Command{
			{
				Name:    "probe-balance-slot",
				Aliases: []string{},
				Usage:   "Probe balance slots",
				Action:  probeBalanceSlotAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "jsonrpcurl-override",
						Usage: "If set, use this URL instead of common.rpc in the configuration file",
					},
					&cli.StringFlag{
						Name:  "wallet",
						Usage: "The wallet address to be probed its balance slot. If not set, a randomized address is used.",
					},
					&cli.BoolFlag{
						Name:  "retry-not-found-tokens",
						Usage: "If set, retry probing tokens that its balance slot is failed to be found",
					},
					&cli.BoolFlag{
						Name:  "skip-existing-tokens",
						Usage: "If set, dont't probe tokens that already exist in Redis (whether balance slot is found or not)",
					},
					&cli.StringSliceFlag{
						Name:  "tokens",
						Usage: "If any, use these tokens instead of loading from Redis",
					},
					&cli.IntFlag{
						Name:  "num-threads",
						Usage: "Number of concurrent probing threads. Default is 1",
						Value: 1,
					},
				},
			},
			{
				Name:    "convert-to-embedded",
				Aliases: []string{},
				Usage:   "Read all balance slots from Redis then convert to embedded format for embedding into router-service.",
				Action:  convertToEmbeddedAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Output file path",
						Required: true,
					},
				},
			},
		}}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func probeBalanceSlotAction(c *cli.Context) error {
	configFile := c.String("config")
	tokenGroupConfigPath := env.StringFromEnv(envvar.TokenGroupConfigPath, "")
	correlatedPairsConfigPath := env.StringFromEnv(envvar.CorrelatedPairsConfigPath, "")

	configLoader, err := config.NewConfigLoader(configFile, []string{tokenGroupConfigPath, correlatedPairsConfigPath})
	if err != nil {
		return err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to validate config")
		panic(err)
	}

	var walletAddr common.Address
	if c.IsSet("wallet") {
		walletAddr = common.HexToAddress(c.String("wallet"))
	} else {
		walletAddr = randomizeAddress()
	}

	var jsonrpcURL string
	if c.IsSet("jsonrpcurl-override") {
		jsonrpcURL = c.String("jsonrpcurl-override")
	} else {
		jsonrpcURL = cfg.Common.RPC
	}

	log.Ctx(c.Context).Info().Msgf("wallet address: %s JSONRPC URL: %s\n", walletAddr, jsonrpcURL)

	retryNotFoundTokens := c.Bool("retry-not-found-tokens")

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		log.Ctx(c.Context).Error().Msg("fail to init redis client to pool service")
		return err
	}

	// get all pools and group by its tokens
	poolsByToken := make(map[common.Address][]*dexentity.Pool)
	key := utils.Join(cfg.PoolRedis.Prefix, pool.KeyPools)
	cursor := uint64(0)
	for {
		keyValues, newCursor, err := poolRedisClient.Client.HScan(context.Background(), key, cursor, "",
			HashKeyReadChunkSize).Result()
		if err != nil {
			return err
		}

		for i := 0; i < len(keyValues); i += 2 {
			p := new(dexentity.Pool)
			// ignore failed to unmarshal pools
			if err := json.Unmarshal([]byte(keyValues[i+1]), p); err != nil {
				continue
			}
			// ignore non-addressable pools
			if !common.IsHexAddress(p.Address) {
				continue
			}

			for _, t := range p.Tokens {
				tokenAddr := common.HexToAddress(t.Address)
				poolsByToken[tokenAddr] = append(poolsByToken[tokenAddr], p)
			}
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	// sort by ReserveUSD descending
	for _, pools := range poolsByToken {
		slices.SortFunc(pools, func(a, b *dexentity.Pool) int { return cmp.Compare(b.ReserveUsd, a.ReserveUsd) })
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client,
		cfg.Repository.ERC20BalanceSlot.Redis)

	tokens := make(map[common.Address]struct{})
	if len(c.StringSlice("tokens")) > 0 {
		for _, t := range c.StringSlice("tokens") {
			tokens[common.HexToAddress(t)] = struct{}{}
		}
	} else {
		tokensList := poolRedisClient.Client.HKeys(context.Background(),
			utils.Join(cfg.PoolRedis.Prefix, token.KeyTokens)).Val()
		for _, t := range tokensList {
			if common.IsHexAddress(t) {
				tokens[common.HexToAddress(t)] = struct{}{}
			}
		}
	}

	balanceSlots, err := balanceSlotRepo.GetAll(context.Background())
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("could not get balance slots")
	}

	if c.Bool("skip-existing-tokens") {
		for t, bl := range balanceSlots {
			if bl.Found {
				delete(tokens, t)
			} else if !retryNotFoundTokens {
				delete(tokens, t)
			}
		}
	}
	log.Ctx(c.Context).Info().Msgf("number of tokens to probe = %v\n", len(tokens))

	rpcClient, err := rpc.DialHTTP(jsonrpcURL)
	if err != nil {
		return err
	}
	probe := erc20balanceslotuc.NewMultipleStrategy(rpcClient, walletAddr)

	var (
		numProbed  atomic.Int64
		numThreads = c.Int("num-threads")
		tokenCh    = make(chan common.Address, numThreads)
		group      = new(errgroup.Group)
	)
	for w := 0; w < numThreads; w++ {
		group.Go(func() error {
			for t := range tokenCh {
				oldBl := balanceSlots[t]
				extraParams := &erc20balanceslotuc.MultipleStrategyExtraParams{}
				if pools, ok := poolsByToken[t]; ok {
					if len(pools) > 0 {
						extraParams.DoubleFromSource = &erc20balanceslotuc.DoubleFromSourceStrategyExtraParams{
							Source: common.HexToAddress(pools[0].Address),
						}
					}
				}
				bl, err := probe.ProbeBalanceSlot(context.Background(), t, oldBl, extraParams)
				if err != nil {
					log.Ctx(c.Context).Err(err).Msg("probe.ProbeBalanceSlot")
				} else {
					log.Ctx(c.Context).Info().Msgf("(%d/%d) %s : %+v\n",
						numProbed.Add(1), len(tokens), t, bl)
					if err := balanceSlotRepo.Put(context.Background(), bl); err != nil {
						return fmt.Errorf("could not PUT: %w", err)
					}
				}
			}
			return nil
		})
	}

	for t := range tokens {
		tokenCh <- t
	}
	close(tokenCh)

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

func randomizeAddress() common.Address {
	a := common.Address{}
	for i := range a {
		a[i] = byte(rand.Intn(256))
	}
	return a
}

func convertToEmbeddedAction(c *cli.Context) error {
	configFile := c.String("config")
	tokenGroupConfigPath := env.StringFromEnv(envvar.TokenGroupConfigPath, "")
	correlatedPairsConfigPath := env.StringFromEnv(envvar.CorrelatedPairsConfigPath, "")

	configLoader, err := config.NewConfigLoader(configFile, []string{tokenGroupConfigPath, correlatedPairsConfigPath})
	if err != nil {
		return err
	}

	cfg, err := configLoader.Get()
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to validate config")
		panic(err)
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		log.Ctx(c.Context).Error().Msg("fail to init redis client to pool service")
		return err
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client, cfg.Repository.ERC20BalanceSlot.Redis)

	bls, err := balanceSlotRepo.GetAll(context.Background())
	if err != nil {
		return err
	}

	serialized, err := erc20balanceslotuc.SerializeEmbedded(bls)
	if err != nil {
		return err
	}
	outputFilePath := c.String("output")
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer func(outputFile *os.File) {
		_ = outputFile.Close()
	}(outputFile)
	if _, err := outputFile.Write(serialized); err != nil {
		return err
	}

	return nil
}
