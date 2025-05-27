package entity

type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals uint8  `json:"decimals"`
}

func (t Token) GetAddress() string {
	return t.Address
}

type SimplifiedToken struct {
	Address  string `json:"address"`
	Decimals uint8  `json:"decimals"`
}

func (t SimplifiedToken) GetAddress() string {
	return t.Address
}
