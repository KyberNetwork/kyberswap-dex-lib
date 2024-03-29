package pancakev3

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueriesPancakeV3_GetPoolsListQuery(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct query when allowing subgraph error", func(t *testing.T) {
		expect := fmt.Sprintf(`{
		pools(
			subgraphError: allow,
			where: {
				createdAtTimestamp_gte: %v
			},
			first: %v,
			skip: %v,
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			feeTier
			token0 {
				id
				name
				symbol
				decimals
			}
			token1 {
				id
				name
				symbol
				decimals
			}
		}
	}`, big.NewInt(0), 1000, 0)

		actual := getPoolsListQuery(true, big.NewInt(0), 1000, 0)

		assert.Equal(t, expect, actual)
	})

	t.Run("it should return correct query when subgraph error is not allowed", func(t *testing.T) {
		expect := fmt.Sprintf(`{
		pools(
			
			where: {
				createdAtTimestamp_gte: %v
			},
			first: %v,
			skip: %v,
			orderBy: createdAtTimestamp,
			orderDirection: asc
		) {
			id
			liquidity
			sqrtPrice
			createdAtTimestamp
			tick
			feeTier
			token0 {
				id
				name
				symbol
				decimals
			}
			token1 {
				id
				name
				symbol
				decimals
			}
		}
	}`, big.NewInt(0), 1000, 0)

		actual := getPoolsListQuery(false, big.NewInt(0), 1000, 0)

		assert.Equal(t, expect, actual)
	})
}

func TestQueriesPancakeV3_GetPoolTicksQuery(t *testing.T) {
	t.Parallel()

	t.Run("it should return correct query when allowing subgraph error", func(t *testing.T) {
		expect := fmt.Sprintf(`{
		ticks(
			subgraphError: allow,
			where: {
				pool: "%v"
				tickIdx_gt: %v,
				liquidityGross_not: 0
			},
			orderBy: tickIdx,
			orderDirection: asc,
			first: 1000
		) {
			tickIdx
			liquidityNet
			liquidityGross
		}
		_meta { block { timestamp }}
	}`, "abc", "0")

		actual := getPoolTicksQuery(true, "abc", "0")

		assert.Equal(t, expect, actual)
	})

	t.Run("it should return correct query when subgraph error is not allowed", func(t *testing.T) {
		expect := fmt.Sprintf(`{
		ticks(
			
			where: {
				pool: "%v"
				tickIdx_gt: %v,
				liquidityGross_not: 0
			},
			orderBy: tickIdx,
			orderDirection: asc,
			first: 1000
		) {
			tickIdx
			liquidityNet
			liquidityGross
		}
		_meta { block { timestamp }}
	}`, "abc", "0")

		actual := getPoolTicksQuery(false, "abc", "0")

		assert.Equal(t, expect, actual)
	})
}
