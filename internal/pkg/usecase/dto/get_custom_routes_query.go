package dto

type GetCustomRoutesQuery struct {
	GetRoutesQuery
	PoolIds        []string
	EnableAlphaFee bool
}
