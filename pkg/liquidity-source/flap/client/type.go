package client

// Coin is the token metadata block inside each board item.
type Coin struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Image   string `json:"image"`
}

// Tax carries the buy/sell tax config for tokens that use flap's tax-token implementation.
type Tax struct {
	HasTax     bool   `json:"hasTax"`
	BuyTaxBps  *int64 `json:"buyTaxBps"`
	SellTaxBps *int64 `json:"sellTaxBps"`
}

// Vault, when non-nil, means the token was launched through a vault factory (out of scope for the
// bonding-curve pool itself, kept for completeness).
type Vault struct {
	Vault        string `json:"vault"`
	VaultFactory string `json:"vaultFactory"`
}

// BoardItem is a single row of GET /v3/board.
type BoardItem struct {
	Coin         Coin   `json:"coin"`
	Listed       bool   `json:"listed"`
	QuoteToken   string `json:"quoteToken"`
	Price        string `json:"price"`
	Progress     string `json:"progress"`
	Tax          Tax    `json:"tax"`
	Vault        *Vault `json:"vault"`
	IsInnovation bool   `json:"isInnovation"`
	IsLowRisk    bool   `json:"isLowRisk"`
	CreatedAt    int64  `json:"createdAt"`
}

// BoardResponse is the payload of GET /v3/board. NextCursor is empty once the last page is reached.
type BoardResponse struct {
	Category   string      `json:"category"`
	Sort       string      `json:"sort"`
	NextCursor string      `json:"nextCursor"`
	Items      []BoardItem `json:"items"`
}
