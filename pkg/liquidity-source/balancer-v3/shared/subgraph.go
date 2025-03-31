package shared

type SubgraphPool struct {
	Address    string `json:"address"`
	CreateTime int64  `json:"createTime"`
	Hook       struct {
		Address string   `json:"address"`
		Type    HookType `json:"type"`
	} `json:"hook"`
	PoolTokens []SubgraphToken `json:"poolTokens"`
}

type SubgraphToken struct {
	Address         string `json:"address"`
	IsErc4626       bool   `json:"isErc4626"`
	UnderlyingToken struct {
		Address string `json:"address"`
	} `json:"underlyingToken"`
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
	$` + VarPoolType + `: GqlPoolType!
	$` + VarCreateTimeGt + `: Int!
	$` + VarFirst + `: Int!
	$` + VarSkip + `: Int!
) {
	poolGetPools(
		where: {
			chainIn: [$` + VarChain + `]
			protocolVersionIn: [3]
			poolTypeIn: [$` + VarPoolType + `]
			createTime: {gt: $` + VarCreateTimeGt + `}
		}
		first: $` + VarFirst + `
		skip: $` + VarSkip + `
	) {
		address
		createTime
		hook {
			address
			type
		}
		poolTokens {
			address
			isErc4626
			underlyingToken {
				address
			}
		}
	}
}`
