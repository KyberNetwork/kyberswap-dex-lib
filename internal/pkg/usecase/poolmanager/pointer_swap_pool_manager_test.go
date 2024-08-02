package poolmanager_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestFilterInvalidPools(t *testing.T) {
	t.Run("it should return success and correctly filter out faulty pool and black list of the input address", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		states := [3]*poolmanager.LockedState{}
		for i := 0; i < 2; i++ {
			states[i] = poolmanager.NewLockedState()
		}
		poolCache, _ := cachePolicy.New[string, struct{}](3)

		poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
		poolRepository := mocks.NewMockIPoolRepository(ctrl)
		poolFactory := mocks.NewMockIPoolFactory(ctrl)

		faultyPools := []string{
			"address1",
			"address2",
		}
		poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools, nil).Times(1)
		addresses := []string{
			faultyPools[0],
			faultyPools[1],
			"address3",
			"address4",
			"address5",
		}

		poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return([]string{"address5"}, nil).Times(1)

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory,
			poolRepository,
			poolRankRepository,
			poolmanager.Config{
				BlacklistedPoolSet: map[string]bool{"address3": true},
			},
			poolCache)

		p.UpdateFaultyPools(context.TODO())
		p.UpdateBlackListPool(context.TODO())
		result := p.FilterInvalidPoolAddresses(addresses)
		expected := []string{"address4"}
		assert.Equal(t, expected, result)
	})

	t.Run("it should return success when faulty pool list return error is empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		faultyError := fmt.Errorf("faulty pool error")

		states := [3]*poolmanager.LockedState{}
		for i := 0; i < 2; i++ {
			states[i] = poolmanager.NewLockedState()
		}
		poolCache, _ := cachePolicy.New[string, struct{}](2)

		poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
		poolRepository := mocks.NewMockIPoolRepository(ctrl)
		poolFactory := mocks.NewMockIPoolFactory(ctrl)

		poolRepository.EXPECT().GetFaultyPools(gomock.Any()).
			Return([]string{}, faultyError).Times(1)

		poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return([]string{"address5", "address6"}, nil).Times(1)

		addresses := []string{
			"address1",
			"address2",
			"address5",
			"address6",
			"address7",
			"address8",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository,
			poolmanager.Config{
				BlacklistedPoolSet: map[string]bool{"address9": true},
			}, poolCache)
		p.UpdateFaultyPools(context.TODO())
		p.UpdateBlackListPool(context.TODO())

		result := p.FilterInvalidPoolAddresses(addresses)
		assert.ElementsMatch(t, result, []string{
			"address1",
			"address2",
			"address7",
			"address8",
		})
	})

	t.Run("it should return the original address list when get faulty pools and black list pool both failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		states := [3]*poolmanager.LockedState{}
		for i := 0; i < 2; i++ {
			states[i] = poolmanager.NewLockedState()
		}
		poolCache, _ := cachePolicy.New[string, struct{}](2)

		poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
		poolRepository := mocks.NewMockIPoolRepository(ctrl)
		poolFactory := mocks.NewMockIPoolFactory(ctrl)

		testError := errors.New("test error")
		poolRepository.EXPECT().GetFaultyPools(gomock.Any()).
			Return([]string{}, testError).Times(1)
		poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return([]string{}, testError).Times(1)

		addresses := []string{
			"address1",
			"address2",
			"address5",
			"address6",
			"address7",
			"address8",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository, poolmanager.Config{}, poolCache)
		p.UpdateFaultyPools(context.TODO())
		p.UpdateBlackListPool(context.TODO())

		result := p.FilterInvalidPoolAddresses(addresses)
		assert.ElementsMatch(t, result, addresses)
	})

}

func TestPointerSwapPoolManager_GetStateByPoolAddresses(t *testing.T) {
	var (
		nTokens = 10
		nPools  = 100
	)
	config := poolmanager.Config{
		StallingPMMThreshold: 500 * time.Millisecond,
		PoolRenewalInterval:  500 * time.Millisecond,
		Capacity:             nPools,
	}

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
	var (
		tokenAddressList = make([]string, len(tokenByAddress))
		i                = 0
	)

	for tokenAddress := range tokenByAddress {
		tokenAddressList[i] = tokenAddress
		i++
	}

	pool1, err := valueobject.GenPMMPool(tokenByAddress[tokenAddressList[0]], tokenByAddress[tokenAddressList[1]])
	require.NoError(t, err)
	pool2, err := valueobject.GenPMMPool(tokenByAddress[tokenAddressList[1]], tokenByAddress[tokenAddressList[2]])
	require.NoError(t, err)
	poolByAddresses := map[string]poolpkg.IPoolSimulator{
		pool1.GetAddress(): pool1,
		pool2.GetAddress(): pool2}

	poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
	poolRepository := mocks.NewMockIPoolRepository(ctrl)
	poolFactory := mocks.NewMockIPoolFactory(ctrl)

	var (
		poolAddressList  = make([]string, len(poolByAddresses))
		poolList         = make([]poolpkg.IPoolSimulator, len(poolByAddresses))
		poolsInBlackList = []string{"none"}
	)
	//reuse index
	i = 0
	for address := range poolByAddresses {
		poolAddressList[i] = address
		poolList[i] = poolByAddresses[address]
		i++
	}
	// Mocked PoolRank always return the poolAddressList above
	poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(poolAddressList, nil).AnyTimes()
	poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(poolsInBlackList, nil).AnyTimes()
	poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return([]string{}, nil).AnyTimes()
	poolRepository.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return([]*entity.Pool{}, nil).AnyTimes()
	poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolList).AnyTimes()
	poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
	poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolByAddresses).AnyTimes()

	pm, err := poolmanager.NewPointerSwapPoolManager(
		context.Background(),
		poolRepository, poolFactory, poolRankRepository, config, nil, nil,
	)
	require.NoError(t, err)
	// let sleep for 3 sec
	time.Sleep(3 * time.Second)
	state, err := pm.GetStateByPoolAddresses(context.Background(), poolAddressList, []string{pooltypes.PoolTypes.KyberPMM}, common.Hash{0x00})
	require.Error(t, err)
	assert.Equal(t, true, state == nil)
}

func TestPointerSwapPoolManager_Start(t *testing.T) {
	t.Skip("flaky test so skip during CI process")
	var (
		nTokens = 10
		nPools  = 30
	)
	config := poolmanager.Config{
		PoolRenewalInterval:        3 * time.Second,
		BlackListRenewalInterval:   4 * time.Second,
		FaultyPoolsRenewalInterval: 2 * time.Second,
		Capacity:                   nPools,
	}

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
	var (
		tokenAddressList = make([]string, len(tokenByAddress))
		i                = 0
	)

	for tokenAddress := range tokenByAddress {
		tokenAddressList[i] = tokenAddress
		i++
	}

	uniswapV3, _ := valueobject.GenerateRandomPoolByAddress(10, tokenAddressList, pooltypes.PoolTypes.UniswapV3)
	curveBase, _ := valueobject.GenerateRandomPoolByAddress(10, tokenAddressList, pooltypes.PoolTypes.CurveBase)
	curveOracle, _ := valueobject.GenerateRandomPoolByAddress(10, tokenAddressList, pooltypes.PoolTypes.CurvePlainOracle)
	poolByAddresses := map[string]poolpkg.IPoolSimulator{}
	maps.Copy(poolByAddresses, uniswapV3)
	maps.Copy(poolByAddresses, curveBase)
	maps.Copy(poolByAddresses, curveOracle)

	poolAddressList := make([]string, 0, len(poolByAddresses))
	for k := range poolByAddresses {
		poolAddressList = append(poolAddressList, k)
	}

	poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
	poolRepository := mocks.NewMockIPoolRepository(ctrl)
	poolFactory := mocks.NewMockIPoolFactory(ctrl)

	// Load all address pool from global best pools with specific capacity 30
	poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(poolAddressList, nil).Times(1)

	// mock poolsInblackList called for every 4 seconds
	poolsInBlackList := []string{poolAddressList[0], poolAddressList[1]}
	poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(poolsInBlackList, nil).Times(2)

	// mock faultyPools called for every 2 seconds
	faultyList := []string{poolAddressList[2], poolAddressList[3]}
	poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyList, nil).Times(3)

	poolEntities := make([]*entity.Pool, 0, len(poolAddressList[4:]))
	for _, addr := range poolAddressList[4:] {
		pool := poolByAddresses[addr]
		poolEntities = append(poolEntities, &entity.Pool{
			Address: pool.GetAddress(),
			Tokens: entity.PoolTokens{
				&entity.PoolToken{Address: pool.GetTokens()[0]},
				&entity.PoolToken{Address: pool.GetTokens()[1]},
			},
			Type: pool.GetType(),
		})
	}
	poolRepository.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return(poolEntities, nil).Times(2)
	numberOfPoolsToInit := 0
	poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolByAddresses).AnyTimes().Do(func(arg0, arg1, arg2 interface{}) {
		numberOfPoolsToInit = len(arg1.([]*entity.Pool))
	})

	pm, err := poolmanager.NewPointerSwapPoolManager(
		context.Background(),
		poolRepository, poolFactory, poolRankRepository, config, nil, nil,
	)
	assert.Equal(t, len(poolAddressList)-len(faultyList)-len(poolsInBlackList), numberOfPoolsToInit)
	require.NoError(t, err)
	// let sleep for 5 sec
	time.Sleep(5 * time.Second)

	// prepares data in start function will be swap pointer from 0 to 1, another swap will happen in goroutine reload
	assert.Equal(t, pm.ReadFrom(), int32(2))
}
