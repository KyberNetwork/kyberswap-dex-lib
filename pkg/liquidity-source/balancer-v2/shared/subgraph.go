package shared

import (
	"fmt"
	"math/big"
	"strings"
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
		Token    struct {
			Pool struct {
				Address string `json:"address"`
			} `json:"pool"`
		} `json:"token"`
	} `json:"tokens"`
}

func BuildSubgraphPoolsQuery(
	poolTypes []string,
	lastCreateTime *big.Int,
	first int,
	skip int,
) string {
	q := `{
		pools(
			where : {
				poolType_in: %v,
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
				token {
					pool {
						address
					}
				}
			}
		}
	}`

	poolTypesStr := fmt.Sprintf(
		`["%v"]`,
		strings.Join(poolTypes, `","`),
	)

	return fmt.Sprintf(q, poolTypesStr, lastCreateTime, first, skip)
}
