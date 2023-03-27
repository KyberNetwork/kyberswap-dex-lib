package validator

func isEthereumAddress(str string) bool {
	return ethAddressRegex.MatchString(str)
}
