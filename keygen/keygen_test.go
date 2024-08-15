package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyGen(t *testing.T) {
	privateKey, publicKey := genPrivatePub()
	publicKeyPEM := marshalPubKey(publicKey)
	assert.Greater(t, len(privateKey), 256)
	assert.Greater(t, len(publicKeyPEM), 256)
}
