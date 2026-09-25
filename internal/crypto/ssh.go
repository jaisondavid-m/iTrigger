package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// KeyPair represents an OpenSSH public and private key pair.
type KeyPair struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

// GenerateSSHKeyPair generates a new Ed25519 SSH key pair formatted for OpenSSH.
func GenerateSSHKeyPair(comment string) (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Generate OpenSSH private key PEM
	privBlock, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	privPEM := string(pem.EncodeToMemory(privBlock))

	// Generate OpenSSH public key
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh public key: %w", err)
	}

	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	if comment != "" {
		pubKeyStr += " " + comment
	}

	return &KeyPair{
		PublicKey:  pubKeyStr,
		PrivateKey: privPEM,
	}, nil
}
