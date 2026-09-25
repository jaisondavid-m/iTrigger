package crypto_test

import (
	"strings"
	"testing"

	"iTrigger/internal/crypto"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	pair, err := crypto.GenerateSSHKeyPair("test-key")
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair failed: %v", err)
	}

	if !strings.HasPrefix(pair.PublicKey, "ssh-ed25519 ") {
		t.Errorf("expected public key to start with ssh-ed25519, got: %s", pair.PublicKey)
	}

	if !strings.HasSuffix(pair.PublicKey, "test-key") {
		t.Errorf("expected public key to include comment 'test-key', got: %s", pair.PublicKey)
	}

	if !strings.Contains(pair.PrivateKey, "BEGIN OPENSSH PRIVATE KEY") {
		t.Errorf("expected private key to be OpenSSH format, got: %s", pair.PrivateKey)
	}
}
