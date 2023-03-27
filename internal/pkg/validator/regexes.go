package validator

import "regexp"

const (
	ethAddressRegexString = `^0x[0-9a-fA-F]{40}$`
)

var (
	ethAddressRegex = regexp.MustCompile(ethAddressRegexString)
)
