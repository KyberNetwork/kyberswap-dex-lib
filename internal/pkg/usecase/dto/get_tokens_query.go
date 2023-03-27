package dto

type GetTokensQuery struct {
	IDs        []string
	PoolTokens bool
	Extra      bool
}
