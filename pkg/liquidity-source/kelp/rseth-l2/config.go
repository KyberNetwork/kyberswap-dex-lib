package rsethl2

type Config struct {
	DexID          string `json:"dexID"`
	LRTDepositPool string `json:"lrtDepositPool"`
	WNative        string `json:"wNative"`
	CheckNative    bool   `json:"checkNative"`
}
