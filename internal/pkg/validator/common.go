package validator

func IsEthereumAddress(str string) bool {
	return ethAddressRegex.MatchString(str)
}
