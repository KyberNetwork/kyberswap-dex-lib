package valueobject

import "math/big"

type Order struct {
	ID                   int64    `json:"id"`
	ChainID              string   `json:"chainId"`
	Salt                 string   `json:"salt"`
	Signature            string   `json:"signature"`
	MakerAsset           string   `json:"makerAsset"`
	TakerAsset           string   `json:"takerAsset"`
	Maker                string   `json:"maker"`
	Receiver             string   `json:"receiver"`
	AllowedSenders       string   `json:"allowedSenders"`
	MakingAmount         *big.Int `json:"makingAmount"`
	TakingAmount         *big.Int `json:"takingAmount"`
	FeeRecipient         string   `json:"feeRecipient"`
	FilledMakingAmount   *big.Int `json:"filledMakingAmount"`
	FilledTakingAmount   *big.Int `json:"filledTakingAmount"`
	MakerTokenFeePercent uint32   `json:"makerTokenFeePercent"`
	MakerAssetData       string   `json:"makerAssetData"`
	TakerAssetData       string   `json:"takerAssetData"`
	GetMakerAmount       string   `json:"getMakerAmount"`
	GetTakerAmount       string   `json:"getTakerAmount"`
	Predicate            string   `json:"predicate"`
	Permit               string   `json:"permit"`
	Interaction          string   `json:"interaction"`
	ExpiredAt            int64    `json:"expiredAt"`
}

type TokenPair struct {
	Token0 string `json:"token0"`
	Token1 string `json:"token1"`
}

type LimitOrderPair struct {
	MakerAsset string `json:"makerAsset"`
	TakerAsset string `json:"takerAsset"`
}
