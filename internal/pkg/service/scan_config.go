package service

import (
	"context"
	"strings"
	"sync"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/repository"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils/envvar"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/util/env"
)

var CoingeckoStaticPrices = map[string]float64{
	"dai":                               1,
	"usd-coin":                          1,
	"binance-usd":                       1,
	"tether":                            1,
	"tether-avalanche-bridged-usdt-e":   1,
	"usd-coin-avalanche-bridged-usdc-e": 1,
	"true-usd":                          1,
	"just-stablecoin":                   1,
	"magic-internet-money":              1,
}

type ScanConfigService struct {
	configLoader      *config.ConfigLoader
	config            *config.Common
	stableTokens      map[string]bool
	whitelistedTokens map[string]bool
	blacklistedPools  map[string]bool
	tokenRepo         repository.ITokenRepository
	priceRepo         repository.IPriceRepository
	mu                sync.RWMutex
}

func NewScanConfigService(
	configLoader *config.ConfigLoader,
	config *config.Common,
	tokenRepo repository.ITokenRepository,
	priceRepo repository.IPriceRepository,
) *ScanConfigService {
	var svc = &ScanConfigService{
		configLoader:      configLoader,
		config:            config,
		stableTokens:      make(map[string]bool),
		whitelistedTokens: make(map[string]bool),
		blacklistedPools:  make(map[string]bool),
		tokenRepo:         tokenRepo,
		priceRepo:         priceRepo,
	}
	err := svc.initTokens()
	if err != nil {
		panic(err)
	}

	svc.initBlacklistedPools()

	return svc
}

func (s *ScanConfigService) ApplyConfig(_ context.Context) error {
	cfg, err := s.configLoader.Get()
	if err != nil {
		return err
	}

	whitelistedTokensByAddress := make(map[string]bool)
	for _, t := range cfg.WhitelistedTokens {
		tokenAddress := strings.ToLower(t.Address)
		whitelistedTokensByAddress[tokenAddress] = true
	}

	s.mu.Lock()
	s.whitelistedTokens = whitelistedTokensByAddress
	s.mu.Unlock()

	return nil
}

func (s *ScanConfigService) initTokens() error {
	ctx := context.Background()

	cfg, err := s.configLoader.Get()
	if err != nil {
		return err
	}

	for _, wt := range cfg.WhitelistedTokens {
		token := entity.Token{
			Address:  strings.ToLower(wt.Address),
			Name:     wt.Name,
			Symbol:   wt.Symbol,
			Decimals: wt.Decimals,
			CgkID:    wt.CgkId,
			Type:     "erc20",
		}

		err := s.tokenRepo.Save(ctx, token)
		if err != nil {
			return err
		}
		price := CoingeckoStaticPrices[token.CgkID]
		if price > 0 {
			price := entity.Price{
				Address:     token.Address,
				Price:       price,
				Liquidity:   99999999999999,
				LpAddress:   token.Address,
				MarketPrice: price,
			}
			s.stableTokens[token.Address] = true
			err = s.priceRepo.Save(ctx, price)
			if err != nil {
				return err
			}
		}
		s.whitelistedTokens[token.Address] = true
	}

	if err != nil {
		logger.Errorf("failed to save tokens: %v", err)
	}

	return err
}

// initBlacklistedPools init the list of pools that we do not want to use to calculate the price
func (s *ScanConfigService) initBlacklistedPools() {
	blacklistedPoolsEnv := env.StringFromEnv(envvar.BlacklistedPools, "")

	if len(blacklistedPoolsEnv) > 0 {
		blacklistedPools := strings.Split(blacklistedPoolsEnv, ",")

		for _, p := range blacklistedPools {
			if len(p) > 0 {
				s.blacklistedPools[p] = true
			}
		}
	}
}

func (s *ScanConfigService) GetWhiteListTokens() map[string]bool {
	return s.whitelistedTokens
}

func (s *ScanConfigService) IsWhiteListToken(address string) bool {
	var _, ok = s.whitelistedTokens[address]
	return ok
}

func (s *ScanConfigService) IsStableToken(address string) bool {
	var _, ok = s.stableTokens[address]
	return ok
}

func (s *ScanConfigService) IsBlacklistedPool(address string) bool {
	var _, ok = s.blacklistedPools[address]
	return ok
}

func (s *ScanConfigService) Config() *config.Common {
	return s.config
}
