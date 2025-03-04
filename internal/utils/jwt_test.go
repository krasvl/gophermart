package utils

import (
	"testing"

	"github.com/krasvl/market/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "mysecret"
	user := storage.User{ID: 1}

	token, err := GenerateToken(user, secret)
	assert.NoError(t, err, "Token generation should not produce an error")
	assert.NotEmpty(t, token, "Generated token should not be empty")

	parsedUserID, err := ParseToken(token, secret)
	assert.NoError(t, err, "Token parsing should not produce an error")
	assert.Equal(t, user.ID, parsedUserID, "Parsed user ID should match the original user ID")
}

func TestParseTokenInvalid(t *testing.T) {
	secret := "mysecret"
	invalidToken := "invalid.token.string"

	_, err := ParseToken(invalidToken, secret)
	assert.Error(t, err, "Parsing an invalid token should produce an error")
}
