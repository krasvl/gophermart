package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidLuhn(t *testing.T) {
	assert.True(t, IsValidLuhn("4532015112830366"), "Valid Luhn number should return true")
	assert.False(t, IsValidLuhn("4532015112830367"), "Invalid Luhn number should return false")
	assert.False(t, IsValidLuhn("1234567890"), "Invalid Luhn number should return false")
	assert.True(t, IsValidLuhn("79927398713"), "Valid Luhn number should return true")
}
