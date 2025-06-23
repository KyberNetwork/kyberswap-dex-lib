package valueobject

type TradeDataType string

const (
	WHITELIST_WHITELIST TradeDataType = "whitelist-whitelist"
	TOKEN_WHITELIST     TradeDataType = "token-whitelist"
	WHITELIST_TOKEN     TradeDataType = "whitelist-token"
	DIRECT              TradeDataType = "direct"
)
