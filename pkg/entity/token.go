package entity

import (
	"github.com/KyberNetwork/kutils"
)

type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals uint8  `json:"decimals"`
}

func (t *Token) GetAddress() string {
	if t == nil {
		return ""
	}
	return t.Address
}

func (t *Token) String() string {
	if t == nil {
		return "nil"
	}
	return "{address:" + t.Address + ",symbol:" + t.Symbol +
		",name:" + t.Name + ",decimals:" + kutils.Utoa(t.Decimals) + "}"
}

type SimplifiedToken struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}

func (t *SimplifiedToken) GetAddress() string {
	if t == nil {
		return ""
	}
	return t.Address
}

func (t *SimplifiedToken) String() string {
	if t == nil {
		return "nil"
	}
	return "{address:" + t.Address + ",decimals:" + kutils.Utoa(t.Decimals) + "}"
}
