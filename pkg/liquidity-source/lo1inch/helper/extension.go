package helper

import (
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

// Extension represents additional data for orders
type Extension struct {
	MakerAssetSuffix string
	TakerAssetSuffix string
	MakingAmountData string
	TakingAmountData string
	Predicate        string
	MakerPermit      string
	PreInteraction   string
	PostInteraction  string
	CustomData       string
}

type ExtensionData struct {
	MakerAssetSuffix string
	TakerAssetSuffix string
	MakingAmountData string
	TakingAmountData string
	Predicate        string
	MakerPermit      string
	PreInteraction   string
	PostInteraction  string
	CustomData       string
}

func NewExtension(data ExtensionData) (*Extension, error) {
	// Validate all fields are hex strings
	fields := map[string]string{
		"MakerAssetSuffix": data.MakerAssetSuffix,
		"TakerAssetSuffix": data.TakerAssetSuffix,
		"MakingAmountData": data.MakingAmountData,
		"TakingAmountData": data.TakingAmountData,
		"Predicate":        data.Predicate,
		"MakerPermit":      data.MakerPermit,
		"PreInteraction":   data.PreInteraction,
		"PostInteraction":  data.PostInteraction,
		"CustomData":       data.CustomData,
	}

	for key, val := range fields {
		if val != ZX && !isHexString(val) {
			return nil, fmt.Errorf("%s must be valid hex string", key)
		}
	}

	return &Extension{
		MakerAssetSuffix: data.MakerAssetSuffix,
		TakerAssetSuffix: data.TakerAssetSuffix,
		MakingAmountData: data.MakingAmountData,
		TakingAmountData: data.TakingAmountData,
		Predicate:        data.Predicate,
		MakerPermit:      data.MakerPermit,
		PreInteraction:   data.PreInteraction,
		PostInteraction:  data.PostInteraction,
		CustomData:       data.CustomData,
	}, nil
}

func DefaultExtension() *Extension {
	ext, _ := NewExtension(ExtensionData{
		MakerAssetSuffix: ZX,
		TakerAssetSuffix: ZX,
		MakingAmountData: ZX,
		TakingAmountData: ZX,
		Predicate:        ZX,
		MakerPermit:      ZX,
		PreInteraction:   ZX,
		PostInteraction:  ZX,
		CustomData:       ZX,
	})
	return ext
}

func (e *Extension) HasPredicate() bool {
	return e.Predicate != ZX
}

func (e *Extension) HasMakerPermit() bool {
	return e.MakerPermit != ZX
}

func DecodeExtension(bytes string) (*Extension, error) {
	if bytes == ZX {
		return DefaultExtension(), nil
	}

	iter := NewBytesIter(bytes)
	offsets, _ := new(big.Int).SetString(strings.TrimPrefix(iter.NextUint256(), "0x"), 16)
	consumed := 0

	fields := []string{
		"MakerAssetSuffix",
		"TakerAssetSuffix",
		"MakingAmountData",
		"TakingAmountData",
		"Predicate",
		"MakerPermit",
		"PreInteraction",
		"PostInteraction",
	}

	data := make(map[string]string)
	mask := new(big.Int).SetUint64(UINT_32_MAX)

	for _, field := range fields {
		offsetBig := new(big.Int).And(offsets, mask)
		offset := int(offsetBig.Int64())
		bytesCount := offset - consumed
		data[field] = iter.NextBytes(bytesCount)

		consumed += bytesCount
		offsets.Rsh(offsets, 32)
	}

	data["CustomData"] = iter.Rest()

	return NewExtension(ExtensionData{
		MakerAssetSuffix: data["MakerAssetSuffix"],
		TakerAssetSuffix: data["TakerAssetSuffix"],
		MakingAmountData: data["MakingAmountData"],
		TakingAmountData: data["TakingAmountData"],
		Predicate:        data["Predicate"],
		MakerPermit:      data["MakerPermit"],
		PreInteraction:   data["PreInteraction"],
		PostInteraction:  data["PostInteraction"],
		CustomData:       data["CustomData"],
	})
}

func (e *Extension) getAll() []string {
	return []string{
		e.MakerAssetSuffix,
		e.TakerAssetSuffix,
		e.MakingAmountData,
		e.TakingAmountData,
		e.Predicate,
		e.MakerPermit,
		e.PreInteraction,
		e.PostInteraction,
		e.CustomData,
	}
}

func (e *Extension) IsEmpty() bool {
	allInteractions := e.getAll()
	allInteractionsConcat := ""
	for _, interaction := range allInteractions {
		allInteractionsConcat += strings.TrimPrefix(interaction, "0x")
	}
	allInteractionsConcat += strings.TrimPrefix(e.CustomData, "0x")

	return len(allInteractionsConcat) == 0
}

func (e *Extension) Keccak256() *big.Int {
	encoded := e.Encode()
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(encoded))
	result := hash.Sum(nil)
	value := new(big.Int).SetBytes(result)
	return value
}

func (e *Extension) Encode() string {
	allInteractions := e.getAll()
	allInteractionsConcat := ""
	for _, interaction := range allInteractions {
		allInteractionsConcat += strings.TrimPrefix(interaction, "0x")
	}

	if len(allInteractionsConcat) == 0 {
		return ZX
	}

	// Calculate offsets for each interaction
	sum := 0
	offsets := new(big.Int)
	for i, interaction := range allInteractions {
		length := len(strings.TrimPrefix(interaction, "0x")) / 2
		if i < len(allInteractions)-1 { // Don't add offset for the last item
			sum += length
			offset := new(big.Int).SetInt64(int64(sum))
			offset.Lsh(offset, uint(32*i))
			offsets.Or(offsets, offset)
		}
	}

	offsetsHex := fmt.Sprintf("%064x", offsets)
	return "0x" + offsetsHex + allInteractionsConcat
}
