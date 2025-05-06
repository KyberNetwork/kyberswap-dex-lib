package nativev1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteParams_ToMap(t *testing.T) {
	tests := []struct {
		name   string
		fields QuoteParams
		want   map[string]string
	}{
		{
			"happy",
			QuoteParams{
				Chain:              "1",
				TokenIn:            "0xdac17f958d2ee523a2206206994597c13d831ec7",
				TokenOut:           "0x6b175474e89094c44da98b954eedeac495271d0f",
				AmountWei:          "1000000000000000000",
				FromAddress:        "0x6b175474e89094c44da98b954eedeac495271d0f",
				BeneficiaryAddress: "0x6b175474e89094c44da98b954eedeac495271d0f",
				ToAddress:          "0x6b175474e89094c44da98b954eedeac495271d0f",
				ExpiryTime:         100,
				Slippage:           "0.1",
			},
			map[string]string{
				"chain":               "1",
				"token_in":            "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"token_out":           "0x6b175474e89094c44da98b954eedeac495271d0f",
				"amount_wei":          "1000000000000000000",
				"from_address":        "0x6b175474e89094c44da98b954eedeac495271d0f",
				"beneficiary_address": "0x6b175474e89094c44da98b954eedeac495271d0f",
				"to_address":          "0x6b175474e89094c44da98b954eedeac495271d0f",
				"expiry_time":         "100",
				"slippage":            "0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &QuoteParams{
				Chain:              tt.fields.Chain,
				TokenIn:            tt.fields.TokenIn,
				TokenOut:           tt.fields.TokenOut,
				AmountWei:          tt.fields.AmountWei,
				FromAddress:        tt.fields.FromAddress,
				BeneficiaryAddress: tt.fields.BeneficiaryAddress,
				ToAddress:          tt.fields.ToAddress,
				ExpiryTime:         tt.fields.ExpiryTime,
				Slippage:           tt.fields.Slippage,
			}
			assert.Equalf(t, tt.want, p.ToMap(), "ToMap()")
		})
	}
}
