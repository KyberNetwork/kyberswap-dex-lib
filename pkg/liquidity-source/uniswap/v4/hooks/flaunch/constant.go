package flaunch

import "github.com/ethereum/go-ethereum/common"

var HookAddresses = []common.Address{
	// PositionManager 1.0
	common.HexToAddress("0x51Bba15255406Cfe7099a42183302640ba7dAFDC"),

	// PositionManager 1.1
	common.HexToAddress("0xF785bb58059FAB6fb19bDdA2CB9078d9E546Efdc"),

	// PositionManager 1.2
	common.HexToAddress("0xB903b0AB7Bcee8f5E4D8C9b10a71aaC7135d6FdC"),

	// PositionManager 1.3
	common.HexToAddress("0x23321f11a6d44Fd1ab790044FdFDE5758c902FDc"),

	// AnyPositionManager 1.0
	common.HexToAddress("0x8DC3b85e1dc1C846ebf3971179a751896842e5dC"),
}
