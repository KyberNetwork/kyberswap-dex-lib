package shared

import (
	"fmt"
	"math/big"
)

type SubgraphPool struct {
	ID              string   `json:"id"`
	Address         string   `json:"address"`
	PoolType        string   `json:"poolType"`
	PoolTypeVersion *big.Int `json:"poolTypeVersion"`
	CreateTime      *big.Int `json:"createTime"`
	Tokens          []struct {
		Address  string `json:"address"`
		Decimals int    `json:"decimals"`
		Weight   string `json:"weight"`
	} `json:"tokens"`
}

func BuildSubgraphPoolsQuery(
	poolType string,
	lastCreateTime *big.Int,
	first int,
	skip int,
) string {
	q := `{
		pools(
			where : {
				poolType: "%v",
				createTime_gte: %v
			},
			first: %v,
			skip: %v,
			orderBy: createTime,
			orderDirection: asc,
		) {
			id
			address
			poolType
			poolTypeVersion
			createTime
			tokens {
			  address
			  decimals
			  weight
			}
		}
	}`

	return fmt.Sprintf(q, poolType, lastCreateTime, first, skip)
}
