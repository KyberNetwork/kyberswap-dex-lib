package poolmanager_test

import (
	"context"
	"errors"
	"sync"
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

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func TestExcludeFaultyPools(t *testing.T) {
	t.Run("it should return success and correctly filter out faulty pool of the input address", func(t *testing.T) {
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

		config := poolmanager.Config{
			FaultyPoolsExpireThreshold: 30 * time.Second,
			MaxFaultyPoolSize:          int64(500),
		}
		faultyPools := []string{
			"address1",
			"address2",
			"address3",
		}
		poolRepository.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Eq(int64(0)), gomock.Eq(config.MaxFaultyPoolSize)).
			Return(faultyPools, nil).Times(1)
		addresses := []string{
			faultyPools[0],
			faultyPools[1],
			"address4",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository, config, poolCache,
			&sync.RWMutex{})
		result := p.ExcludeFaultyPools(context.Background(), addresses, config)
		expected := []string{"address4"}
		assert.Equal(t, result, expected)
	})

	t.Run("it should return success and correctly filter out faulty pool of the input address with paging mechanism", func(t *testing.T) {
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

		config := poolmanager.Config{
			FaultyPoolsExpireThreshold: 30 * time.Second,
			MaxFaultyPoolSize:          int64(4),
		}
		faultyPools := []string{
			"address1",
			"address2",
			"address3",
			"address4",
		}
		poolRepository.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Eq(int64(0)), gomock.Eq(config.MaxFaultyPoolSize)).
			Return(faultyPools, nil).Times(1)
		poolRepository.EXPECT().GetFaultyPools(
			gomock.Any(),
			gomock.Any(),
			gomock.Eq(config.MaxFaultyPoolSize),
			gomock.Eq(config.MaxFaultyPoolSize)).
			Return([]string{
				"address5",
				"address6",
			}, nil).Times(1)

		addresses := []string{
			"address1",
			"address2",
			"address5",
			"address6",
			"address7",
			"address8",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository, config, poolCache,
			&sync.RWMutex{})
		result := p.ExcludeFaultyPools(context.Background(), addresses, config)
		expected := []string{"address7", "address8"}
		assert.ElementsMatch(t, result, expected)
	})

	t.Run("it should return success when faulty pool list is empty", func(t *testing.T) {
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

		config := poolmanager.Config{
			FaultyPoolsExpireThreshold: 30 * time.Second,
			MaxFaultyPoolSize:          int64(500),
		}
		poolRepository.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Eq(int64(0)), gomock.Eq(config.MaxFaultyPoolSize)).
			Return([]string{}, nil).Times(1)

		addresses := []string{
			"address1",
			"address2",
			"address5",
			"address6",
			"address7",
			"address8",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository, config, poolCache,
			&sync.RWMutex{})
		result := p.ExcludeFaultyPools(context.Background(), addresses, config)
		assert.ElementsMatch(t, result, addresses)
	})

	t.Run("it should return the original address list when get faulty pools failed", func(t *testing.T) {
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

		config := poolmanager.Config{
			FaultyPoolsExpireThreshold: 30 * time.Second,
			MaxFaultyPoolSize:          int64(500),
		}
		testError := errors.New("test error")
		poolRepository.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Eq(int64(0)), gomock.Eq(config.MaxFaultyPoolSize)).
			Return([]string{}, testError).Times(1)

		addresses := []string{
			"address1",
			"address2",
			"address5",
			"address6",
			"address7",
			"address8",
		}

		p := poolmanager.NewPointerSwapPoolManagerInstance(
			states, poolFactory, poolRepository, poolRankRepository, config, poolCache,
			&sync.RWMutex{})
		result := p.ExcludeFaultyPools(context.Background(), addresses, config)
		assert.ElementsMatch(t, result, addresses)
	})

}

func TestPointerSwapPoolManager_GetStateByPoolAddresses(t *testing.T) {
	var (
		nTokens = 10
		nPools  = 100
	)
	config := poolmanager.Config{
		StallingPMMThreshold:       500 * time.Millisecond,
		PoolRenewalInterval:        500 * time.Millisecond,
		FaultyPoolsExpireThreshold: 30 * time.Second,
		MaxFaultyPoolSize:          int64(500),
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

	pool1, err := valueobject.GenPMMPool(tokenByAddress[tokenAddressList[0]], tokenByAddress[tokenAddressList[1]])
	require.NoError(t, err)
	pool2, err := valueobject.GenPMMPool(tokenByAddress[tokenAddressList[1]], tokenByAddress[tokenAddressList[2]])
	require.NoError(t, err)
	poolByAddresses := map[string]poolpkg.IPoolSimulator{
		pool1.GetAddress(): pool1,
		pool2.GetAddress(): pool2}

	//poolCache, _ := cachePolicy.New[string, struct{}](2)
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
	poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(poolAddressList).AnyTimes()
	poolRepository.EXPECT().PoolsInBlacklist(gomock.Any()).Return(poolsInBlackList, nil).AnyTimes()
	poolRepository.EXPECT().GetFaultyPools(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]string{}, nil).AnyTimes()
	poolRepository.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return([]*entity.Pool{}, nil).AnyTimes()
	poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolList).AnyTimes()
	poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
	poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolByAddresses).AnyTimes()

	pm, err := poolmanager.NewPointerSwapPoolManager(
		context.Background(),
		poolRepository, poolFactory, poolRankRepository, config, nil,
	)
	require.NoError(t, err)
	// let sleep for 2 sec
	time.Sleep(3 * time.Second)
	state, err := pm.GetStateByPoolAddresses(context.Background(), poolAddressList, []string{pooltypes.PoolTypes.KyberPMM}, common.Hash{0x00})
	require.NoError(t, err)
	_, pool1Avail := state.Pools[pool1.GetAddress()]
	assert.Equal(t, false, pool1Avail)
	_, pool2Avail := state.Pools[pool1.GetAddress()]
	assert.Equal(t, false, pool2Avail)
}
