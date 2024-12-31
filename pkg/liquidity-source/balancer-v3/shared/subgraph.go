package shared

import (
	"fmt"
	"math/big"
)

type SubgraphPool struct {
	ID             string `json:"id"`
	Address        string `json:"address"`
	BlockTimestamp string `json:"blockTimestamp"`
	Factory        string `json:"factory"`
	Tokens         []struct {
		Address  string `json:"address"`
		Decimals int    `json:"decimals"`
	} `json:"tokens"`
	Vault struct {
		ID string `json:"id"`
	} `json:"vault"`
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
			factory
			tokens {
			  address
			  decimals
			}
			vault {
			  id
			}
		}
	}`

	return fmt.Sprintf(q, lastBlockTimestamp, first, skip)
}
