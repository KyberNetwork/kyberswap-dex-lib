package saddle

type PoolStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}
type Extra struct {
	InitialA     string `json:"initialA"`
	FutureA      string `json:"futureA"`
	InitialATime int64  `json:"initialATime"`
	FutureATime  int64  `json:"futureATime"`
	SwapFee      string `json:"swapFee"`
	AdminFee     string `json:"adminFee"`
	//LpToken            string `json:"lpToken"`
	DefaultWithdrawFee string `json:"defaultWithdrawFee"`
}

type Meta struct {
	TokenInIndex  int `json:"tokenInIndex"`
	TokenOutIndex int `json:"tokenOutIndex"`
	PoolLength    int `json:"poolLength"`
}

type Gas struct {
	Swap            int64
	AddLiquidity    int64
	RemoveLiquidity int64
}
