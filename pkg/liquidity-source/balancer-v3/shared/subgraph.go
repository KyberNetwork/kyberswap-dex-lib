package shared

import (
	"fmt"
	"math/big"
)

type SubgraphPool struct {
	ID         string   `json:"id"`
	Address    string   `json:"address"`
	CreateTime *big.Int `json:"blockTimestamp"`
	Tokens     []struct {
		Address       string `json:"address"`
		Decimals      int    `json:"decimals"`
		ScalingFactor string `json:"scalingFactor"`
	} `json:"tokens"`
}

func BuildSubgraphPoolsQuery(
	lastBlockTimestamp *big.Int,
	first int,
	skip int,
) string {
	q := `{
		pools(
			where : {
				blockTimestamp_gte: %v
			},
			first: %d,
			skip: %d,
			orderBy: blockTimestamp,
			orderDirection: asc,
		) {
			id
			address
			blockTimestamp
			tokens {
			  address
			  decimals
			  scalingFactor
			}
		}
	}`

	return fmt.Sprintf(q, lastBlockTimestamp, first, skip)
}
