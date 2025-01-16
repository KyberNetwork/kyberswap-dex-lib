package shared

import (
	"fmt"
	"math/big"
	"strings"
)

type SubgraphPool struct {
	ID             string `json:"id"`
	Address        string `json:"address"`
	BlockTimestamp string `json:"blockTimestamp"`
	Tokens         []struct {
		Address  string `json:"address"`
		Decimals int    `json:"decimals"`
	} `json:"tokens"`
	Vault struct {
		ID string `json:"id"`
	} `json:"vault"`
}

func BuildSubgraphPoolsQuery(
	factory string,
	lastBlockTimestamp *big.Int,
	first int,
	skip int,
) string {
	q := `{
		pools(
			where : {
				factory: "%s",
				blockTimestamp_gte: %v
				symbol_not: "TEST"
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
			}
			vault {
			  id
			}
		}
	}`

	return fmt.Sprintf(q, strings.ToLower(factory), lastBlockTimestamp, first, skip)
}
