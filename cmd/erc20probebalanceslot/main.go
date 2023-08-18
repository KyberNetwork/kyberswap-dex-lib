package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
	erc20balanceslotuc "github.com/KyberNetwork/router-service/internal/pkg/usecase/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
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

	configLoader, err := config.NewConfigLoader(configFile)
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
		logger.Errorf("failed to validate config, err: %v", err)
		panic(err)
	}

	var walletAddr common.Address
	if c.IsSet("wallet") {
		walletAddr = common.HexToAddress(c.String("wallet"))
	} else {
		walletAddr = randomizeAddress()
	}

	fmt.Printf("wallet address: %s\n", walletAddr)

	var jsonrpcURL string
	if c.IsSet("jsonrpcurl-override") {
		jsonrpcURL = c.String("jsonrpcurl-override")
	} else {
		jsonrpcURL = cfg.Common.RPC
	}

	fmt.Printf("JSONRPC URL: %s\n", jsonrpcURL)

	retryNotFoundTokens := c.Bool("retry-not-found-tokens")

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf("fail to init redis client to pool service")
		return err
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client, erc20balanceslot.RedisRepositoryConfig{
		Prefix: cfg.PoolRedis.Prefix,
	})

	var tokens []string
	if len(c.StringSlice("tokens")) > 0 {
		tokens = c.StringSlice("tokens")
	} else {
		tokens = poolRedisClient.Client.HKeys(context.Background(), utils.Join(cfg.PoolRedis.Prefix, token.KeyTokens)).Val()
	}
	fmt.Printf("numTokens = %v\n", len(tokens))

	skippedTokens := make(map[common.Address]struct{})
	if c.Bool("skip-existing-tokens") {
		balanceSlots, err := balanceSlotRepo.GetAll(context.Background())
		if err != nil {
			logger.Errorf("could not get balance slots %s", err)
		}

		for token, bl := range balanceSlots {
			if bl.Found {
				skippedTokens[token] = struct{}{}
			} else if !retryNotFoundTokens {
				skippedTokens[token] = struct{}{}
			}
		}
	}

	rpcClient, err := rpc.DialHTTP(jsonrpcURL)
	if err != nil {
		return err
	}
	probe := erc20balanceslotuc.NewProbe(rpcClient, walletAddr)

	var newBalanceSlots []*entity.ERC20BalanceSlot
	for i, token := range tokens {
		if _, ok := skippedTokens[common.HexToAddress(token)]; ok {
			continue
		}

		slot, err := probe.ProbeBalanceSlot(common.HexToAddress(token))
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			newBalanceSlots = append(newBalanceSlots, &entity.ERC20BalanceSlot{
				Token:  token,
				Wallet: walletAddr.String(),
				Found:  false,
			})
		} else {
			fmt.Printf("(%d/%d) %s : %s\n", i+1, len(tokens), token, slot)
			newBalanceSlots = append(newBalanceSlots, &entity.ERC20BalanceSlot{
				Token:       token,
				Wallet:      walletAddr.String(),
				Found:       true,
				BalanceSlot: slot.String(),
			})
		}
	}

	return balanceSlotRepo.PutMany(context.Background(), newBalanceSlots)
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

	configLoader, err := config.NewConfigLoader(configFile)
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
		logger.Errorf("failed to validate config, err: %v", err)
		panic(err)
	}

	poolRedisClient, err := redis.New(&cfg.PoolRedis)
	if err != nil {
		logger.Errorf("fail to init redis client to pool service")
		return err
	}

	balanceSlotRepo := erc20balanceslot.NewRedisRepository(poolRedisClient.Client, erc20balanceslot.RedisRepositoryConfig{
		Prefix: cfg.PoolRedis.Prefix,
	})

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
