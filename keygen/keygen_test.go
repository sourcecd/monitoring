package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyGen(t *testing.T) {
	privateKey, publicKey, err := genPrivatePub()
	require.NoError(t, err)
	publicKeyPEM, err := marshalPubKey(publicKey)
	require.NoError(t, err)
	assert.Greater(t, len(privateKey), 256)
	assert.Greater(t, len(publicKeyPEM), 256)
}
