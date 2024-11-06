package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"slices"
	"sync/atomic"

	dexentity "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	erc20balanceslotuc "github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/router-service/pkg/util/env"
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
				Name:    "convert-to-preloaded",
				Aliases: []string{},
				Usage:   "Read all balance slots from Redis then convert to preloaded format for embedding into router-service.",
				Action:  convertToPreloadedAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Output file path",
						Required: true,
					},
				},
			},
		}}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
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

	_, err = logger.InitLogger(cfg.Log.Configuration, logger.LoggerBackendZap)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		logger.Errorf(c.Context, "failed to validate config, err: %v", err)
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

	logger.Infof(c.Context, "wallet address: %s JSONRPC URL: %s\n", walletAddr, jsonrpcURL)

	retryNotFoundTokens := c.Bool("retry-not-found-tokens")

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(c.Context, "fail to init redis client to pool service")
		return err
	}

	// get all pools and group by its tokens
	rawPools := poolRedisClient.Client.HGetAll(context.Background(), utils.Join(cfg.PoolRedis.Prefix, pool.KeyPools)).Val()
	poolsByToken := make(map[common.Address][]*dexentity.Pool)
	for _, rawPool := range rawPools {
		pool := new(dexentity.Pool)
		// ignore failed to unmarshal pools
		if err := json.Unmarshal([]byte(rawPool), pool); err != nil {
			continue
		}
		// ignore non-addressable pools
		if !common.IsHexAddress(pool.Address) {
			continue
		}
		for _, token := range pool.Tokens {
			tokenAddr := common.HexToAddress(token.Address)
			poolsByToken[tokenAddr] = append(poolsByToken[tokenAddr], pool)
		}
	}
	// sort by ReserveUSD descending
	for _, pools := range poolsByToken {
		slices.SortFunc(pools, func(a, b *dexentity.Pool) int { return cmp.Compare(b.ReserveUsd, a.ReserveUsd) })
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client, cfg.Repository.ERC20BalanceSlot.Redis)

	tokens := make(map[common.Address]struct{})
	if len(c.StringSlice("tokens")) > 0 {
		for _, token := range c.StringSlice("tokens") {
			tokens[common.HexToAddress(token)] = struct{}{}
		}
	} else {
		tokensList := poolRedisClient.Client.HKeys(context.Background(), utils.Join(cfg.PoolRedis.Prefix, token.KeyTokens)).Val()
		for _, token := range tokensList {
			if common.IsHexAddress(token) {
				tokens[common.HexToAddress(token)] = struct{}{}
			}
		}
	}

	balanceSlots, err := balanceSlotRepo.GetAll(context.Background())
	if err != nil {
		logger.Errorf(c.Context, "could not get balance slots %s", err)
	}

	if c.Bool("skip-existing-tokens") {
		for token, bl := range balanceSlots {
			if bl.Found {
				delete(tokens, token)
			} else if !retryNotFoundTokens {
				delete(tokens, token)
			}
		}
	}
	logger.Infof(c.Context, "number of tokens to probe = %v\n", len(tokens))

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
			for token := range tokenCh {
				oldBl := balanceSlots[token]
				extraParams := &erc20balanceslotuc.MultipleStrategyExtraParams{}
				if pools, ok := poolsByToken[token]; ok {
					if len(pools) > 0 {
						extraParams.DoubleFromSource = &erc20balanceslotuc.DoubleFromSourceStrategyExtraParams{
							Source: common.HexToAddress(pools[0].Address),
						}
					}
				}
				bl, err := probe.ProbeBalanceSlot(context.Background(), token, oldBl, extraParams)
				if err != nil {
					logger.Infof(c.Context, "ERROR: %s\n", err)
				} else {
					logger.Infof(c.Context, "(%d/%d) %s : %+v\n", numProbed.Add(1), len(tokens), token, bl)
					if err := balanceSlotRepo.Put(context.Background(), bl); err != nil {
						return fmt.Errorf("could not PUT: %w", err)
					}
				}
			}
			return nil
		})
	}

	for token := range tokens {
		tokenCh <- token
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

func convertToPreloadedAction(c *cli.Context) error {
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

	_, err = logger.InitLogger(cfg.Log.Configuration, logger.LoggerBackendZap)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		logger.Errorf(c.Context, "failed to validate config, err: %v", err)
		panic(err)
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf(c.Context, "fail to init redis client to pool service")
		return err
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client, cfg.Repository.ERC20BalanceSlot.Redis)

	bls, err := balanceSlotRepo.GetAll(context.Background())
	if err != nil {
		return err
	}

	serialized, err := erc20balanceslotuc.SerializePreloaded(bls)
	if err != nil {
		return err
	}
	outputFilePath := c.String("output")
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	if _, err := outputFile.Write(serialized); err != nil {
		return err
	}

	return nil
}
