package vault

import (
	"context"

	"github.com/Indellient/vault-helper/pkg/logger"
)

// Creates, validates, and initializes a new Client with specified params
func NewVaultClient(ctx context.Context, addr string, insecure bool) *Client {
	vault := new(Client)
	vault.Address = addr
	vault.Insecure = insecure
	vault.ctx = ctx

	// Basic validation of input
	err := vault.Validate()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	// Extended validation of input -- can we actually communicate with vault?
	err = vault.ExtendedValidate()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	return vault
}
