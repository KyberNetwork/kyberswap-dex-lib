package service

import (
	"context"
	"testing"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func TestScanService_calculateAmplifiedTvl(t *testing.T) {
	ctx := context.Background()
	scanService := ScanService{}

	t.Run("correct AmplifiedTvl when token0 or token1 equal 0 (token0 equal 0)", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"0", "1"},
			Type:       constant.PoolTypes.Dodo,
			ReserveUsd: float64(11),
		})

		assert.EqualValues(t, 0, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})

	t.Run("correct AmplifiedTvl when token0 or token1 equal 0 (token1 equal 0)", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"1", "0"},
			Type:       constant.PoolTypes.Lido,
			ReserveUsd: float64(99),
		})

		assert.EqualValues(t, 0, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})

	t.Run("correct AmplifiedTvl when token0 and token1 equal 0", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"0", "0"},
			Type:       constant.PoolTypes.Synthetix,
			ReserveUsd: float64(1),
		})

		assert.EqualValues(t, 0, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})

	t.Run("correct AmplifiedTvl when token1 and token2 are greater than 0(reserveUSD = 100)", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"1", "1"},
			Type:       constant.PoolTypes.Lido,
			ReserveUsd: float64(100),
		})

		assert.EqualValues(t, 100, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})

	t.Run("correct AmplifiedTvl when token1 and token2 are greater than 0(reserveUSD = 0)", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"999999999", "1"},
			Type:       constant.PoolTypes.Uni,
			ReserveUsd: float64(0),
		})

		assert.EqualValues(t, 0, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})

	t.Run("correct AmplifiedTvl when token1 and token2 are greater than 0(reserveUSD = 999999999)", func(t *testing.T) {
		amplifiedTvl, err := scanService.calculateAmplifiedTvl(ctx, entity.Pool{
			Reserves:   []string{"999999999", "999999999"},
			Type:       constant.PoolTypes.Uni,
			ReserveUsd: float64(999999999),
		})

		assert.EqualValues(t, 999999999, amplifiedTvl)
		assert.ErrorIs(t, err, nil)
	})
}
