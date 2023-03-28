package makerpsm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var ErrMakerPSMPoolNotFound = errors.New("no maker-psm pool found")

type Scanner struct {
	scanService *service.ScanService
	scanDexCfg  *config.ScanDex
	properties  ScannerProperties

	psmReader IPSMReader
	vatReader IVatReader
}

func New(
	scanDexCfg *config.ScanDex,
	scanService *service.ScanService,
) (core.IScanDex, error) {
	properties, err := NewScannerProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}

	return &Scanner{
		scanService: scanService,
		scanDexCfg:  scanDexCfg,
		properties:  properties,

		psmReader: NewPSMReader(scanService),
		vatReader: NewVatReader(scanService),
	}, nil
}

func (s *Scanner) InitPool(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		logger.Infof("initialized pool in %v", time.Since(startTime))
	}()

	config, err := s.getConfig()
	if err != nil {
		return err
	}

	for _, psmConfig := range config.PSMs {
		psm, err := s.getPsm(ctx, psmConfig.Address)
		if err != nil {
			return err
		}

		pool, err := s.newPool(psm, psmConfig, config.Dai)
		if err != nil {
			return err
		}

		s.scanService.SavePool(ctx, *pool)

		for _, token := range pool.Tokens {
			if _, err = s.scanService.FetchOrGetToken(ctx, token.Address); err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateNewPools does nothing,
// all pools are configured in config file and already initialized in InitPool
func (s *Scanner) UpdateNewPools(ctx context.Context) {}

// UpdateReserves update
func (s *Scanner) UpdateReserves(ctx context.Context) {
	for {
		if err := s.updateReserves(ctx); err != nil {
			logger.Errorf("updateReserves failed, error: %v", err)
		}

		time.Sleep(s.properties.ReserveJobInterval.Duration)
	}
}

// UpdateTotalSupply does nothing
func (s *Scanner) UpdateTotalSupply(ctx context.Context) {}

func (s *Scanner) getConfig() (*Config, error) {
	configFilePath := path.Join(
		s.scanService.Config().DataFolder,
		s.properties.ConfigPath,
	)

	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}

	defer configFile.Close()

	configFileContent, err := io.ReadAll(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = json.Unmarshal(configFileContent, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *Scanner) getPsm(ctx context.Context, address string) (*PSM, error) {
	psm, err := s.psmReader.Read(ctx, address)
	if err != nil {
		return nil, err
	}

	vat, err := s.vatReader.Read(ctx, psm.VatAddress.String(), psm.ILK)
	if err != nil {
		return nil, err
	}

	psm.Vat = vat

	return psm, nil
}

func (s *Scanner) newPool(psm *PSM, config PSMConfig, daiConfig DaiConfig) (*entity.Pool, error) {
	token0 := &entity.PoolToken{
		Address:   daiConfig.Address,
		Decimals:  daiConfig.Decimals,
		Swappable: true,
	}

	reserve0 := new(big.Int).Sub(
		new(big.Int).Div(
			psm.Vat.ILK.Line,
			psm.Vat.ILK.Rate,
		),
		psm.Vat.ILK.Art,
	)

	token1 := &entity.PoolToken{
		Address:   config.Gem.Address,
		Decimals:  config.Gem.Decimals,
		Swappable: true,
	}

	reserve1 := psm.Vat.ILK.Art

	extra := struct {
		PSM *PSM `json:"psm"`
	}{
		PSM: psm,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	return &entity.Pool{
		Address:   strings.ToLower(config.Address),
		Exchange:  s.scanDexCfg.Id,
		Type:      constant.PoolTypes.MakerPSM,
		Tokens:    []*entity.PoolToken{token0, token1},
		Reserves:  []string{reserve0.String(), reserve1.String()},
		Extra:     string(extraBytes),
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *Scanner) updateReserves(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		executionTime := time.Since(startTime)

		logger.
			WithFields(logger.Fields{
				"dex":               s.scanDexCfg.Id,
				"poolsUpdatedCount": 1,
				"duration":          executionTime.Milliseconds(),
			}).
			Info("finished UpdateReserves")

		metrics.HistogramScannerUpdateReservesDuration(executionTime, s.scanDexCfg.Id, 1)
	}()

	pools, err := s.scanService.GetPoolsByExchange(ctx, s.scanDexCfg.Id)
	if err != nil {
		return err
	}

	if len(pools) == 0 {
		return ErrMakerPSMPoolNotFound
	}

	for _, pool := range pools {
		psm, err := s.getPsm(ctx, pool.Address)
		if err != nil {
			return err
		}

		extra := struct {
			PSM *PSM `json:"psm"`
		}{
			PSM: psm,
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return err
		}

		reserve0 := new(big.Int).Sub(
			new(big.Int).Div(
				psm.Vat.ILK.Line,
				psm.Vat.ILK.Rate,
			),
			psm.Vat.ILK.Art,
		)

		reserve1 := psm.Vat.ILK.Art

		pool.Reserves = []string{reserve0.String(), reserve1.String()}
		pool.Extra = string(extraBytes)
		pool.Timestamp = time.Now().Unix()

		if err := s.scanService.SavePool(ctx, pool); err != nil {
			return fmt.Errorf("save pool failed, err: %v", err)
		}
	}

	return nil
}
