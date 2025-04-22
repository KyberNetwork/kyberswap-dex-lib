package shared

import (
	"fmt"
	"strings"
)

type NestedToken struct {
	Address string `json:"address"`
}

type NestedPool struct {
	Address string        `json:"address"`
	Tokens  []NestedToken `json:"tokens"`
}

type PoolToken struct {
	Address    string     `json:"address"`
	IsAllowed  bool       `json:"isAllowed"`
	Weight     string     `json:"weight"`
	Decimals   int        `json:"decimals"`
	NestedPool NestedPool `json:"nestedPool"`
}

type SubgraphPool struct {
	ID         string      `json:"id"`
	Address    string      `json:"address"`
	CreateTime int64       `json:"createTime"`
	Type       string      `json:"type"`
	Version    int         `json:"version"`
	PoolTokens []PoolToken `json:"poolTokens"`
}

type SubgraphPoolV1 struct {
	ID              string `json:"id"`
	Address         string `json:"address"`
	PoolType        string `json:"poolType"`
	PoolTypeVersion int    `json:"poolTypeVersion"`
	CreateTime      int64  `json:"createTime"`
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

const (
	VarChain        = "chain"
	VarPoolType     = "poolType"
	VarCreateTimeGt = "createTimeGt"
	VarFirst        = "first"
	VarSkip         = "skip"
)

const SubgraphPoolsQueryV2 = `query(
	$` + VarChain + `: GqlChain!
	$` + VarPoolType + `: [GqlPoolType!]!
	$` + VarCreateTimeGt + `: Int!
	$` + VarFirst + `: Int!
	$` + VarSkip + `: Int!
) {
	poolGetPools(
		where: {
			chainIn: [$` + VarChain + `]
			protocolVersionIn: [2]
			poolTypeIn: $` + VarPoolType + `
			createTime: {gt: $` + VarCreateTimeGt + `}
		}
		first: $` + VarFirst + `
		skip: $` + VarSkip + `
	) {
		id
		address
		createTime
		type
		version
		poolTokens {
			address
      		isAllowed
			weight
			decimals
			nestedPool {
				address
				tokens{
					address
				}
      		}
		}
	}
}`

const SubgraphPoolsQueryBerachain = `query(
	$` + VarChain + `: GqlChain!
	$` + VarPoolType + `: [GqlPoolType!]!
	$` + VarCreateTimeGt + `: Int!
	$` + VarFirst + `: Int!
	$` + VarSkip + `: Int!
) {
	poolGetPools(
		where: {
			chainIn: [$` + VarChain + `]
			protocolVersionIn: [2]
			poolTypeIn: $` + VarPoolType + `
			createTime: {gt: $` + VarCreateTimeGt + `}
		}
		first: $` + VarFirst + `
		skip: $` + VarSkip + `
	) {
		id
		address
		createTime
		type
		version
		allTokens {
			address
			weight
			decimals
		}
	}
}`

func BuildSubgraphPoolsQueryV1(
	poolTypes []string,
	lastCreateTime int64,
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
			}
		}
	}`

	poolTypesStr := fmt.Sprintf(
		`["%v"]`,
		strings.Join(poolTypes, `","`),
	)

	return fmt.Sprintf(q, poolTypesStr, lastCreateTime, first, skip)
}
