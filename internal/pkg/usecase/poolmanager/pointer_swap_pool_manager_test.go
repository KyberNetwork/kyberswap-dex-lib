package poolmanager_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	cachePolicy "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/assert"

	mocks "github.com/KyberNetwork/router-service/internal/pkg/mocks/poolmanager"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/poolmanager"
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
