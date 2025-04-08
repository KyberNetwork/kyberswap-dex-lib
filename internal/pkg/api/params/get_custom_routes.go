package params

type GetCustomRoutesParams struct {
	GetRoutesParams
	PoolIds        string `form:"poolIds"`
	EnableAlphaFee bool   `form:"enableAlphaFee"`
}
