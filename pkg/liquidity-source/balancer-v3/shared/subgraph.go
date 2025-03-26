package shared

type SubgraphPool struct {
	Address        string          `json:"address"`
	BlockTimestamp int64           `json:"blockTimestamp,string"`
	IsInitialized  bool            `json:"isInitialized"`
	Tokens         []SubgraphToken `json:"tokens"`
	Vault          struct {
		ID string `json:"id"`
	} `json:"vault"`
}

type SubgraphToken struct {
	Address string `json:"address"`
	Buffer  *struct {
		UnderlyingToken struct {
			ID string `json:"id"`
		} `json:"underlyingToken"`
	} `json:"buffer"`
}

const (
	VarFactory           = "factory"
	VarBlockTimestampGte = "blockTimestampGte"
	VarFirst             = "first"
)

const SubgraphPoolsQuery = `query(
	$` + VarFactory + `: Bytes
	$` + VarBlockTimestampGte + `: BigInt
	$` + VarFirst + `: Int
) {
	pools(
		where : {
			factory: $` + VarFactory + `
			blockTimestamp_gte: $` + VarBlockTimestampGte + `
		}
		first: $` + VarFirst + `
		orderBy: blockTimestamp
		orderDirection: asc
	) {
		address
		blockTimestamp
		isInitialized
		tokens {
			address
			buffer {
				underlyingToken {
					id
				}
			}
		}
		vault {
			id
		}
	}
}`
