package getroute

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCache_GetBestRouteFromCache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		param        *types.AggregateParams
		keys         []*valueobject.RouteCacheKeyTTL
		cachedRoutes map[*valueobject.RouteCacheKeyTTL]*valueobject.SimpleRoute
		bestRoute    *valueobject.SimpleRoute
		err          error
	}{
		{
			name: "",
			param: &types.AggregateParams{
				TokenIn: entity.Token{
					Decimals: 18,
				},
				AmountIn:        bigIntFromScientificNotation("200e18"),
				TokenInPriceUSD: 1,
			},
			keys: newMultiRouteCacheKeys([]float64{250, 198, 215}, valueobject.RouteCacheModeRangeByAmount, []time.Duration{40 * time.Second, 40 * time.Second, 40 * time.Second}),
			cachedRoutes: map[*valueobject.RouteCacheKeyTTL]*valueobject.SimpleRoute{
				{
					Key: &valueobject.RouteCacheKey{
						CacheMode: string(valueobject.RouteCacheModeRangeByAmount),
						AmountIn:  strconv.FormatFloat(250, 'f', 0, 64),
					},
				}: {
					Distributions: []uint64{250},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "a", TokenOutAddress: "b", PoolAddress: "0xabc"}},
					},
				},
				{
					Key: &valueobject.RouteCacheKey{
						CacheMode: string(valueobject.RouteCacheModeRangeByAmount),
						AmountIn:  strconv.FormatFloat(198, 'f', 0, 64),
					},
				}: {
					Distributions: []uint64{198},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
					},
				},
				{
					Key: &valueobject.RouteCacheKey{
						CacheMode: string(valueobject.RouteCacheModeRangeByAmount),
						AmountIn:  strconv.FormatFloat(215, 'f', 0, 64),
					},
				}: {
					Distributions: []uint64{215},
					Paths: [][]valueobject.SimpleSwap{
						{{TokenInAddress: "m", TokenOutAddress: "n", PoolAddress: "0xlmnop"}},
					},
				},
			},
			bestRoute: &valueobject.SimpleRoute{
				Distributions: []uint64{198},
				Paths: [][]valueobject.SimpleSwap{
					{{TokenInAddress: "x", TokenOutAddress: "y", PoolAddress: "0xxyz"}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			routeRepo := mocks.NewMockIRouteCacheRepository(ctrl)
			routeRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(tc.cachedRoutes, tc.err)

			cache := NewCache(nil, routeRepo, nil, valueobject.CacheConfig{})
			_, bestRoute, err := cache.getBestRouteFromCache(context.Background(), tc.param, tc.keys)

			assert.Equal(t, bestRoute, tc.bestRoute)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}
