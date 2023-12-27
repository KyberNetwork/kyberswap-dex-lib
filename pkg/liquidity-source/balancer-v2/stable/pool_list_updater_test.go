package stable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getPoolSpecialization(t *testing.T) {
	t.Run("1. should return correct answer", func(t *testing.T) {
		r, err := _getPoolSpecialization("0x851523a36690bf267bbfec389c823072d82921a90002000000000000000001ed")
		assert.Nil(t, err)
		assert.Equal(t, r, uint8(2))
	})

	t.Run("1. should return correct answer", func(t *testing.T) {
		r, err := _getPoolSpecialization("0xa6f548df93de924d73be7d25dc02554c6bd66db500020000000000000000000e")
		assert.Nil(t, err)
		assert.Equal(t, r, uint8(2))
	})

	t.Run("1. should return correct answer", func(t *testing.T) {
		r, err := _getPoolSpecialization("0x79c58f70905f734641735bc61e45c19dd9ad60bc0000000000000000000004e7")
		assert.Nil(t, err)
		assert.Equal(t, r, uint8(0))
	})
}
