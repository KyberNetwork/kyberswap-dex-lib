package poolmanager_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/poolmanager"
	getPoolsMocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/usecase/getpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getpools"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolfactory"
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
			poolCache, mapset.NewSet[string](), mapset.NewSet[string]())

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
			}, poolCache, mapset.NewSet[string](), mapset.NewSet[string]())
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
			states, poolFactory, poolRepository, poolRankRepository,
			poolmanager.Config{}, poolCache, mapset.NewSet[string](), mapset.NewSet[string]())
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
	getPoolsUsecase := mocks.NewMockIGetPoolsIncludingBasePools(ctrl)

	var (
		poolAddressList  = make([]string, len(poolByAddresses))
		poolList         = make([]poolpkg.IPoolSimulator, len(poolByAddresses))
		poolsInBlackList = []string{"none"}
		whitelistDexes   = []string{pooltypes.PoolTypes.KyberPMM}
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
	getPoolsUsecase.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*entity.Pool{}, nil).AnyTimes()
	poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolList).AnyTimes()
	poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
	poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolByAddresses).AnyTimes()

	pm, err := poolmanager.NewPointerSwapPoolManager(
		context.Background(),
		poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
	)
	require.NoError(t, err)
	// let sleep for 3 sec
	time.Sleep(3 * time.Second)
	state, err := pm.GetStateByPoolAddresses(context.Background(), poolAddressList, whitelistDexes, common.Hash{0x00})
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

	poolRepoUc := getPoolsMocks.NewMockIPoolRepository(ctrl)
	getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUc)

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
	poolRepoUc.EXPECT().FindByAddresses(gomock.Any(), gomock.Any()).Return(poolEntities, nil).Times(2)
	numberOfPoolsToInit := 0
	poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(poolByAddresses).AnyTimes().Do(func(arg0, arg1, arg2 interface{}) {
		numberOfPoolsToInit = len(arg1.([]*entity.Pool))
	})

	pm, err := poolmanager.NewPointerSwapPoolManager(
		context.Background(),
		poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
	)
	assert.Equal(t, len(poolAddressList)-len(faultyList)-len(poolsInBlackList), numberOfPoolsToInit)
	require.NoError(t, err)
	// let sleep for 5 sec
	time.Sleep(5 * time.Second)

	// prepares data in start function will be swap pointer from 0 to 1, another swap will happen in goroutine reload
	assert.Equal(t, pm.ReadFrom(), int32(2))
}

func loadPoolsFromFile(fileName string) []*entity.Pool {
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	res := []*entity.Pool{}
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		pool := new(entity.Pool)
		err := json.Unmarshal([]byte(line), &pool)
		if err != nil {
			fmt.Printf("error scanning input from file line %v error %v\n", line, err)
			continue
		}
		res = append(res, pool)
		i++
	}

	return res
}

func TestPointerSwapPoolManager_GetStateByPoolAddressesTest(t *testing.T) {
	// Setup to load pool state from start up
	var (
		nTokens = 60
		// nPools is total pool = pool_redis + pool_mem
		nPools = 48
	)
	config := poolmanager.Config{
		StallingPMMThreshold: 500 * time.Second,
		PoolRenewalInterval:  5 * time.Second,
		Capacity:             nPools,
		AvailableSources:     []string{"uniswap", "sushiswap", "pancake", "uniswapv3", "curve", "curve-stable-ng", "curve-stable-meta-ng", "curve-tricrypto-ng", "curve-stable-plain", "curve-twocrypto-ng", "smardex"},
	}

	tokenByAddress := valueobject.GenerateRandomTokenByAddress(nTokens)
	var (
		tokenAddressList = make([]string, len(tokenByAddress))
		i                = 0
	)
	for tokenAddress := range tokenByAddress {
		tokenAddressList[i] = tokenAddress
		i++
	}

	// init pools that are available in memory, they are cached locally
	memPoolSimulatorsByAddresses := map[string]poolpkg.IPoolSimulator{}
	memPools := loadPoolsFromFile("./pools_mem.txt")
	for _, p := range memPools {
		assert.NotNil(t, p)
	}

	poolEntitiesByAddress := map[string]*entity.Pool{}
	for _, p := range memPools {
		poolEntitiesByAddress[p.Address] = p
	}
	factory := poolfactory.NewPoolFactory(poolfactory.Config{
		ChainID: 1,
		UseAEVM: false,
	}, nil, nil)
	memPoolSimulators := factory.NewPools(context.TODO(), memPools, common.Hash{})
	for _, si := range memPoolSimulators {
		memPoolSimulatorsByAddresses[si.GetAddress()] = si
	}
	for _, p := range poolEntitiesByAddress {
		if _, ok := memPoolSimulatorsByAddresses[p.Address]; !ok {
			fmt.Printf("init failed %s\n", p.Address)
		}
	}
	memPoolAddressList := make([]string, len(memPoolSimulators))
	i = 0
	for address := range memPoolSimulatorsByAddresses {
		memPoolAddressList[i] = address
		i++
	}

	// init some mock pools that are not available in memory
	redisPoolSimulatorsByAddresses := map[string]poolpkg.IPoolSimulator{}
	redisPools := loadPoolsFromFile("./pools_redis.txt")
	for _, p := range redisPools {
		assert.NotNil(t, p)
	}
	poolOnRedisEntitiesByAddress := map[string]*entity.Pool{}
	for _, p := range redisPools {
		poolOnRedisEntitiesByAddress[p.Address] = p
	}
	redisPoolSimulators := factory.NewPools(context.TODO(), redisPools, common.Hash{})
	for _, si := range redisPoolSimulators {
		redisPoolSimulatorsByAddresses[si.GetAddress()] = si
	}

	testCases := []struct {
		name                  string
		inputPoolAddr         mapset.Set[string]
		dex                   mapset.Set[string]
		expectedPoolAddresses mapset.Set[string]
		prepare               func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager
		blacklist             mapset.Set[string]
		faultyPools           mapset.Set[string]
		err                   error
	}{
		{
			name:                  "Fetch base bool of curve-stable-meta-ng successfully from Redis",
			inputPoolAddr:         mapset.NewThreadUnsafeSet("0xb71edd5322ce0309dc30f07d25470dbfcb275c28", "0x2482dfb5a65d901d137742ab1095f26374509352"),
			dex:                   mapset.NewThreadUnsafeSet(pooltypes.PoolTypes.KyberPMM, "uniswap", "curve-stable-meta-ng"),
			expectedPoolAddresses: mapset.NewThreadUnsafeSet("0xb71edd5322ce0309dc30f07d25470dbfcb275c28", "0x2482dfb5a65d901d137742ab1095f26374509352", "0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// mock data for getPools usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0x2482dfb5a65d901d137742ab1095f26374509352"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"]}, nil).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			blacklist:   mapset.NewThreadUnsafeSet[string](),
			faultyPools: mapset.NewThreadUnsafeSet[string](),
		},
		{
			name:                  "Ignore pools are not included in whitelist dex set",
			inputPoolAddr:         mapset.NewThreadUnsafeSet("0x2482dfb5a65d901d137742ab1095f26374509352", "0x896f67c8966a13b50496190acf91bb761ab742e8", "0xb51ca0cbbcd58498d3f104134332786e63480b81"),
			dex:                   mapset.NewThreadUnsafeSet(pooltypes.PoolTypes.KyberPMM, "smardex"),
			expectedPoolAddresses: mapset.NewThreadUnsafeSet("0x896f67c8966a13b50496190acf91bb761ab742e8"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// Fetch pools in pool uscase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0x2482dfb5a65d901d137742ab1095f26374509352"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			blacklist:   mapset.NewThreadUnsafeSet[string](),
			faultyPools: mapset.NewThreadUnsafeSet[string](),
		},
		{
			name:                  "Ignore pools (0xba77add57760da34fbbe205fc868440fd69da2d6) and (0x9e10f9fb6f0d32b350cee2618662243d4f24c64a) are included in faultyPools",
			inputPoolAddr:         mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81", "0xba77add57760da34fbbe205fc868440fd69da2d6", "0x9e10f9fb6f0d32b350cee2618662243d4f24c64a"),
			dex:                   mapset.NewThreadUnsafeSet(pooltypes.PoolTypes.KyberPMM, "uniswap", "curve-stable-meta-ng"),
			expectedPoolAddresses: mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// mock data for getPools usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			faultyPools: mapset.NewThreadUnsafeSet("0xba77add57760da34fbbe205fc868440fd69da2d6"),
			blacklist:   mapset.NewThreadUnsafeSet("0x9e10f9fb6f0d32b350cee2618662243d4f24c64a"),
		},
		{
			name:                  "All pools are available in memory, no need to fetch from Redis",
			inputPoolAddr:         mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81", "hashflow_v3_mm21_1_0x514910771af9ca656af840dff83e8264ecf986ca_0xdac17f958d2ee523a2206206994597c13d831ec7", "swaap_v2_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2_0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			dex:                   mapset.NewThreadUnsafeSet("hashflow-v3", "uniswap"),
			expectedPoolAddresses: mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81", "hashflow_v3_mm21_1_0x514910771af9ca656af840dff83e8264ecf986ca_0xdac17f958d2ee523a2206206994597c13d831ec7"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// mock data for pool usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			blacklist:   mapset.NewThreadUnsafeSet[string](),
			faultyPools: mapset.NewThreadUnsafeSet[string](),
		},
		{
			name:                  "Returns only pools available in memory, pools from Redis can't be fetched due to errors",
			inputPoolAddr:         mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81", "0xba77add57760da34fbbe205fc868440fd69da2d6", "0x7fc77b5c7614e1533320ea6ddc2eb61fa00a9714"),
			dex:                   mapset.NewThreadUnsafeSet("hashflow-v3", "uniswap"),
			expectedPoolAddresses: mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// mock data for get pools usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder([]string{"0xba77add57760da34fbbe205fc868440fd69da2d6", "0x7fc77b5c7614e1533320ea6ddc2eb61fa00a9714"})).Return(
					nil,
					errors.New("some error")).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			blacklist:   mapset.NewThreadUnsafeSet[string](),
			faultyPools: mapset.NewThreadUnsafeSet[string](),
		},
		{
			name:          "Returns ErrPoolSetEmpty due to all pools are in Redis but get failed",
			inputPoolAddr: mapset.NewThreadUnsafeSet("0xba77add57760da34fbbe205fc868440fd69da2d6", "0x9e10f9fb6f0d32b350cee2618662243d4f24c64a", "0x326290a1b0004eee78fa6ed4f1d8f4b2523ab669"),
			dex:           mapset.NewThreadUnsafeSet("pancake", "curve-stable-meta-ng"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				// Mock data for get pools usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(memPoolAddressList)).Return(memPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder([]string{"0xba77add57760da34fbbe205fc868440fd69da2d6", "0x9e10f9fb6f0d32b350cee2618662243d4f24c64a", "0x326290a1b0004eee78fa6ed4f1d8f4b2523ab669"})).Return(
					nil,
					errors.New("some error")).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			blacklist:   mapset.NewThreadUnsafeSet[string](),
			faultyPools: mapset.NewThreadUnsafeSet[string](),
			err:         getroute.ErrPoolSetEmpty,
		},
		{
			name:          "Returns ErrPoolSetFiltered due to all pools are included in faulty list, blacklist and not in whitelist dexes",
			inputPoolAddr: mapset.NewThreadUnsafeSet("0xb51ca0cbbcd58498d3f104134332786e63480b81", "0x9e10f9fb6f0d32b350cee2618662243d4f24c64a", "0x326290a1b0004eee78fa6ed4f1d8f4b2523ab669", "0x9e96fbaa1b6be45111194a8757f0448897f93229"),
			dex:           mapset.NewThreadUnsafeSet("kyber-pmm"),
			prepare: func(ctrl *gomock.Controller, blacklist, faultyPools mapset.Set[string]) *poolmanager.PointerSwapPoolManager {
				poolRankRepository := mocks.NewMockIPoolRankRepository(ctrl)
				poolFactory := mocks.NewMockIPoolFactory(ctrl)

				// Fetch pool states, only fetch pools in pools_mem.txt file
				poolRankRepository.EXPECT().FindGlobalBestPools(gomock.Any(), gomock.Any()).Return(memPoolAddressList, nil).AnyTimes()

				poolRepoUsecase := getPoolsMocks.NewMockIPoolRepository(ctrl)
				getPoolsUsecase := getpools.NewGetPoolsIncludingBasePools(poolRepoUsecase)

				poolFactory.EXPECT().NewPools(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(arg0, arg1, arg2 interface{}) []poolpkg.IPoolSimulator {
					poolEntities := arg1.([]*entity.Pool)
					res := []poolpkg.IPoolSimulator{}
					for _, p := range poolEntities {
						if simulator, ok := memPoolSimulatorsByAddresses[p.Address]; ok {
							res = append(res, simulator)
						} else {
							res = append(res, redisPoolSimulatorsByAddresses[p.Address])
						}
					}
					return res
				}).AnyTimes()
				poolFactory.EXPECT().NewSwapLimit(gomock.Any()).Return(map[string]poolpkg.SwapLimit{}).AnyTimes()
				poolFactory.EXPECT().NewPoolByAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(memPoolSimulatorsByAddresses).AnyTimes()

				poolRepository := mocks.NewMockIPoolRepository(ctrl)

				// Fetch state for faulty pools and blacklist
				poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(blacklist.ToSlice(), nil).AnyTimes()
				poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(faultyPools.ToSlice(), nil).AnyTimes()

				filteredMemAddresses := lo.Filter[string](memPoolAddressList, func(p string, _ int) bool {
					return !blacklist.ContainsOne(p) && !faultyPools.ContainsOne(p)
				})
				filteredMemPools := lo.Filter(memPools, func(p *entity.Pool, _ int) bool {
					return !blacklist.ContainsOne(p.Address) && !faultyPools.ContainsOne(p.Address)
				})

				// Mock data for get pools usecase
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.InAnyOrder(filteredMemAddresses)).Return(filteredMemPools, nil).Times(1)
				// fetch base pool for curve family pools
				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7"})).Return([]*entity.Pool{poolOnRedisEntitiesByAddress["0x2482dfb5a65d901d137742ab1095f26374509352"]}, nil).Times(1)

				poolRepoUsecase.EXPECT().FindByAddresses(gomock.Any(), gomock.Eq([]string{"0x9e10f9fb6f0d32b350cee2618662243d4f24c64a"})).Return(
					[]*entity.Pool{
						poolOnRedisEntitiesByAddress["0x9e10f9fb6f0d32b350cee2618662243d4f24c64a"],
					}, nil).Times(1)

				pm, err := poolmanager.NewPointerSwapPoolManager(
					context.Background(),
					poolRepository, poolFactory, poolRankRepository, getPoolsUsecase, config, nil, nil, nil,
				)
				require.NoError(t, err)

				return pm
			},
			faultyPools: mapset.NewThreadUnsafeSet("0x326290a1b0004eee78fa6ed4f1d8f4b2523ab669"),
			blacklist:   mapset.NewThreadUnsafeSet("0x9e96fbaa1b6be45111194a8757f0448897f93229"),
			err:         getroute.ErrPoolSetFiltered,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ctrl = gomock.NewController(t)
			defer ctrl.Finish()

			pm := tc.prepare(ctrl, tc.blacklist, tc.faultyPools)
			state, err := pm.GetStateByPoolAddresses(context.Background(), tc.inputPoolAddr.ToSlice(), tc.dex.ToSlice(), common.Hash{0x00})

			// verify result
			if state != nil {
				assert.Equal(t, tc.expectedPoolAddresses.Cardinality(), len(state.Pools))
				resultAddress := mapset.NewThreadUnsafeSet[string]()
				for _, p := range state.Pools {
					resultAddress.Add(p.GetAddress())
				}
				assert.Equal(t, resultAddress, tc.expectedPoolAddresses)
			}
			if tc.err == nil {
				assert.Nil(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}

}

func TestPointerSwapPoolManager_UpdateBlacklist(t *testing.T) {
	testcases := []struct {
		name         string
		oldBlacklist mapset.Set[string]
		newBlacklist []string
	}{
		{
			name:         "Update blacklist pools correctly",
			oldBlacklist: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newBlacklist: []string{"0xabc", "0xdef", "0xgkh"},
		},
		{
			name:         "Update blacklist pools correctly when new set is empty",
			oldBlacklist: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newBlacklist: []string{},
		},
		{
			name:         "Update blacklist pools correctly when new set is the same with old set",
			oldBlacklist: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newBlacklist: []string{"0xabc", "0xdef", "0xxyz"},
		},
		{
			name:         "Update blacklist pools correctly when new set has more items than old set",
			oldBlacklist: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newBlacklist: []string{"0xabc", "0xdef", "0xxyz", "0xgkh", "0xlmn"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var ctrl = gomock.NewController(t)
			defer ctrl.Finish()

			poolRepository := mocks.NewMockIPoolRepository(ctrl)
			poolRepository.EXPECT().GetPoolsInBlacklist(gomock.Any()).Return(tc.newBlacklist, nil)

			states := [3]*poolmanager.LockedState{}
			for i := 0; i < 2; i++ {
				states[i] = poolmanager.NewLockedState()
			}
			pm := poolmanager.NewPointerSwapPoolManagerInstance(
				states, nil, poolRepository, nil,
				poolmanager.Config{
					BlacklistedPoolSet: map[string]bool{"address3": true},
				}, nil, nil, tc.oldBlacklist)
			pm.UpdateBlackListPool(context.TODO())
			assert.ElementsMatch(t, pm.BlackListPool().ToSlice(), tc.newBlacklist)
		})
	}
}

func TestPointerSwapPoolManager_UpdateFaultyPools(t *testing.T) {
	testcases := []struct {
		name           string
		oldFaultyPools mapset.Set[string]
		newFaultyPools []string
	}{
		{
			name:           "Update faulty pools correctly",
			oldFaultyPools: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newFaultyPools: []string{"0xabc", "0xdef", "0xgkh"},
		},
		{
			name:           "Update faulty pools correctly when new set is empty",
			oldFaultyPools: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newFaultyPools: []string{},
		},
		{
			name:           "Update faulty pools correctly when new set is the same with old set",
			oldFaultyPools: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newFaultyPools: []string{"0xabc", "0xdef", "0xxyz"},
		},
		{
			name:           "Update faulty pools correctly when new set has more items than old set",
			oldFaultyPools: mapset.NewSet("0xabc", "0xdef", "0xxyz"),
			newFaultyPools: []string{"0xabc", "0xdef", "0xxyz", "0xgkh", "0xlmn"},
		},
		{
			name:           "Update faulty pools correctly when the old set is empty",
			oldFaultyPools: mapset.NewSet[string](),
			newFaultyPools: []string{"0xabc", "0xdef", "0xxyz", "0xgkh", "0xlmn"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var ctrl = gomock.NewController(t)
			defer ctrl.Finish()

			poolRepository := mocks.NewMockIPoolRepository(ctrl)
			poolRepository.EXPECT().GetFaultyPools(gomock.Any()).Return(tc.newFaultyPools, nil)

			states := [3]*poolmanager.LockedState{}
			for i := 0; i < 2; i++ {
				states[i] = poolmanager.NewLockedState()
			}
			pm := poolmanager.NewPointerSwapPoolManagerInstance(
				states, nil, poolRepository, nil, poolmanager.Config{}, nil, tc.oldFaultyPools, nil)
			pm.UpdateFaultyPools(context.TODO())
			assert.ElementsMatch(t, pm.FaultyPools().ToSlice(), tc.newFaultyPools)
		})
	}
}
