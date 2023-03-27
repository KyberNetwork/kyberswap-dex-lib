package params

type GetTokensParams struct {
	IDs        string `form:"ids"`
	PoolTokens bool   `form:"poolTokens"`
	Extra      bool   `form:"extra"`
}
