package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	assert.Equal(t, Join("polygon", "tokenamounts"), "polygon:tokenamounts")
}
