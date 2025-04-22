package shared

type SubgraphPool struct {
	ID         string `json:"id"`
	Address    string `json:"address"`
	CreateTime int64  `json:"createTime"`
	Type       string `json:"type"`
	Version    int    `json:"version"`
	PoolTokens []struct {
		Address    string `json:"address"`
		IsAllowed  bool   `json:"isAllowed"`
		Weight     string `json:"weight"`
		Decimals   int    `json:"decimals"`
		NestedPool struct {
			Address string `json:"address"`
			Tokens  []struct {
				Address string `json:"address"`
			} `json:"tokens"`
		} `json:"nestedPool"`
	} `json:"poolTokens"`
}

const (
	VarChain        = "chain"
	VarPoolType     = "poolType"
	VarCreateTimeGt = "createTimeGt"
	VarFirst        = "first"
	VarSkip         = "skip"
)

const SubgraphPoolsQuery = `query(
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
