package lending

type (
	Assets struct {
		Address  string
		Decimals uint8
		Symbol   string
	}

	LendingVault struct {
		AmmAddress string
		Assets     map[string]Assets
	}

	GetLendingVaultsResult struct {
		Success bool
		Data    struct {
			LendingVaultData []LendingVault
		}
	}
)
