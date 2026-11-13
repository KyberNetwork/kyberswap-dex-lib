package utils

import "github.com/ethereum/go-ethereum/common"

const (
	AddressHalfLength = common.AddressLength / 2
)

type AddressHalf [AddressHalfLength]byte

func HalfAddressFromAddress(a common.Address) AddressHalf {
	var addressHalf AddressHalf
	copy(addressHalf[:], a.Bytes()[common.AddressLength-AddressHalfLength:]) // take the last 10 bytes
	return addressHalf
}

func AddressFromFirstBytes(s []byte) common.Address {
	return common.BytesToAddress(s[:common.AddressLength])
}

func BytesToAddressHalf(bs []byte) AddressHalf {
	var addressHalf AddressHalf
	copy(addressHalf[:], bs)
	return addressHalf
}

func HexToAddressHalf(hex string) AddressHalf {
	return BytesToAddressHalf(common.FromHex(hex))
}
