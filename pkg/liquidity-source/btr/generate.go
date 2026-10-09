//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --abi abis/Pool.json --pkg btr --type Pool --out gen_pool.go
//go:generate sh -c "cp ../../../../dex-evm/abi/Pool.json abis/Pool.json"

package btr
