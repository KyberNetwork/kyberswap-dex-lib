package quoting

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

type PoolKey struct {
	Token0 common.Address `json:"token0"`
	Token1 common.Address `json:"token1"`
	Config Config         `json:"config"`

	stringId string
	numId    *big.Int
}

type AbiPoolKey struct {
	Token0 common.Address
	Token1 common.Address
	Config [32]byte
}

func NewPoolKey(token0, token1 common.Address, config Config) PoolKey {
	return PoolKey{
		Token0: token0,
		Token1: token1,
		Config: config,
	}
}

func (k *PoolKey) StringId() string {
	if k.stringId == "" {
		k.stringId = k.Token0.Hex() + "/" + k.Token1.Hex() + "_" + strconv.FormatUint(k.Config.Fee, 10) + "_" + strconv.FormatUint(uint64(k.Config.TickSpacing), 10) + "_" + k.Config.Extension.Hex()
	}

	return k.stringId
}

func (k *PoolKey) NumId() (*big.Int, error) {
	if k.numId == nil {
		addressTy, _ := abi.NewType("address", "address", nil)
		bytes32Ty, _ := abi.NewType("bytes32", "bytes32", nil)

		enc, err := abi.Arguments{
			{Type: addressTy}, {Type: addressTy}, {Type: bytes32Ty},
		}.Pack(
			k.Token0,
			k.Token1,
			[32]byte(k.Config.Compressed()),
		)
		if err != nil {
			return nil, fmt.Errorf("computing numerical id: %w", err)
		}

		hash := sha3.NewLegacyKeccak256()

		if _, err := hash.Write(enc); err != nil {
			return nil, fmt.Errorf("computing digest: %w", err)
		}

		k.numId = new(big.Int).SetBytes(hash.Sum(nil))
	}

	return k.numId, nil
}

func (k *PoolKey) ToAbi() AbiPoolKey {
	return AbiPoolKey{
		Token0: k.Token0,
		Token1: k.Token1,
		Config: [32]byte(k.Config.Compressed()),
	}
}

type Config struct {
	Fee         uint64         `json:"fee"`
	TickSpacing uint32         `json:"tickSpacing"`
	Extension   common.Address `json:"extension"`

	compressed []byte
}

func (c *Config) Compressed() []byte {
	if c.compressed == nil {
		c.compressed = append(c.compressed, c.Extension.Bytes()...)
		c.compressed = binary.BigEndian.AppendUint64(c.compressed, c.Fee)
		c.compressed = binary.BigEndian.AppendUint32(c.compressed, c.TickSpacing)
	}

	return c.compressed
}
