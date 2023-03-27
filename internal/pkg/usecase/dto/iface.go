package dto

import "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"

// IGetTokensResultTokenBuilder owns logic of building GetTokensResult
type IGetTokensResultTokenBuilder interface {
	// WithToken builds token related data
	WithToken(token entity.Token) IGetTokensResultTokenBuilder
	// WithPrice builds price related data
	WithPrice(price entity.Price) IGetTokensResultTokenBuilder
	// WithPool builds pool related data
	WithPool(pool entity.Pool, tokenByAddress map[string]entity.Token) IGetTokensResultTokenBuilder
	// GetToken returns built GetTokensResultToken
	GetToken() *GetTokensResultToken
}
