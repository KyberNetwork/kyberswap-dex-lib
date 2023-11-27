package subgraph

import (
	"fmt"
	"math/big"
)

type Pool struct {
	ID              string   `json:"id"`
	Address         string   `json:"address"`
	PoolType        string   `json:"poolType"`
	PoolTypeVersion int      `json:"poolTypeVersion"`
	CreateTime      *big.Int `json:"createTime"`
	Tokens          []struct {
		Address string `json:"address"`
		Weight  string `json:"weight"`
	} `json:"tokens"`
}

func GetPoolsQuery(
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
			createTime
			tokens {
			  address
			  weight
			}
		}
	}`

	return fmt.Sprintf(q, poolType, lastCreateTime, first, skip)
}
