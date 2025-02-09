package spark

type Config struct {
	DexID        string `json:"dexID"`
	DepositToken string `json:"depositToken"`
	SavingsToken string `json:"savingsToken"`
	Pot          string `json:"pot"`
	// ssr (Sky Savings Rate) | dsr (DAI Savings Rate)
	SavingsRateSymbol string `json:"savingsRateSymbol"`
}
