package vault_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"vault"
)

var (
	invalidAddrTests = []string{
		"google.com",
		"http//google.com",
		"https//:google.com",
	}

	validAddrTests = []string{
		"http://google.com",
		"https://google.com",
		"https://google.com:8200",
	}
)

func Setup(address, roleId, secretId, token, path, file, selector string) *vault.Client {
	return &vault.Client{
		Address:  address,
		RoleId:   roleId,
		SecretId: secretId,
		Token:    token,
		Path:     path,
		File:     file,
		Selector: selector,
	}
}

func TestClient_Validate(t *testing.T) {
	// Invalid addr tests
	for _, v := range invalidAddrTests {
		client := Setup(v, "", "", "", "", "", "")
		assert.NotNil(t, client.Validate(), "Expected Validate() to return error for address '%v'", v)
	}

	// Valid addr tests
	for _, v := range validAddrTests {
		client := Setup(v, "", "", "", "", "", "")
		assert.Nil(t, client.Validate(), "Expected Validate() to return nil for address '%v': %v", v, client.Validate())
	}
}

func TestClient_ValidateCreateToken(t *testing.T) {
	// Our client var
	var client *vault.Client

	// Missing secret id
	client = Setup("https://google.com", "dead-beef", "", "", "", "", "")
	assert.NotNil(t, client.ValidateCreateToken(), "Expected ValidateCreateToken() to return error for empty secret id")

	// Missing role id
	client = Setup("https://google.com", "", "ea7-beef", "", "", "", "")
	assert.NotNil(t, client.ValidateCreateToken(), "Expected ValidateCreateToken() to return error for empty role id")

	// Missing role id and secret id
	client = Setup("https://google.com", "", "", "", "", "", "")
	assert.NotNil(t, client.ValidateCreateToken(), "Expected ValidateCreateToken() to return error for empty secret and role id")

	// Valid role id and secret id
	client = Setup("https://google.com", "dead-beef", "ea7-beef", "", "", "", "")
	assert.Nil(t, client.ValidateCreateToken(), "Expected ValidateCreateToken() to return nil for valid role and secret id: %v", client.ValidateCreateToken())
}

func TestClient_ValidateRenewToken(t *testing.T) {
	// Our client var
	var client *vault.Client

	// Missing token
	client = Setup("https://google.com", "", "", "", "", "", "")
	assert.NotNil(t, client.ValidateRenewToken(), "Expected ValidateRenewToken() to return error for empty token")

	// Valid token
	client = Setup("https://google.com", "", "", "dead-c0de", "", "", "")
	assert.Nil(t, client.ValidateRenewToken(), "Expected ValidateRenewToken() to return nil for valid token")
}

func TestClient_ValidateRevokeToken(t *testing.T) {
	// Our client var
	var client *vault.Client

	// Missing token
	client = Setup("https://google.com", "", "", "", "", "", "")
	assert.NotNil(t, client.ValidateRevokeToken(), "Expected ValidateRevokeToken() to return error for empty token")

	// Valid token
	client = Setup("https://google.com", "", "", "dead-c0de", "", "", "")
	assert.Nil(t, client.ValidateRevokeToken(), "Expected ValidateRevokeToken() to return nil for valid token")
}

func TestClient_ValidateFetchSecret(t *testing.T) {
	// Our client var
	var client *vault.Client

	// Missing token
	client = Setup("https://google.com", "", "", "", "/foo/bar", "", "((.username))")
	assert.NotNil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return error for empty token")

	// Missing path
	client = Setup("https://google.com", "", "", "dead-c0de", "", "", "((.username))")
	assert.NotNil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return error for empty path")

	// Missing selector
	client = Setup("https://google.com", "", "", "dead-c0de", "/foo/bar", "", "")
	assert.NotNil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return error for empty selector")

	// Invalid selector
	client = Setup("https://google.com", "", "", "dead-c0de", "/foo/bar", "", "((.username")
	assert.NotNil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return error for invalid selector '((.username'")

	// Missing token, path, and selector
	client = Setup("https://google.com", "", "", "", "", "", "")
	assert.NotNil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return error for missing token, path, and selector")

	// Valid token, path, and selector
	client = Setup("https://google.com", "", "", "dead-c0de", "/foo/bar", "", "((.username))")
	assert.Nil(t, client.ValidateFetchSecret(), "Expected ValidateFetchSecret() to return nil for valid token, path, and selector: %v", client.ValidateFetchSecret())
}

func TestClient_ValidateParseFile(t *testing.T) {
	// Our client var
	var client *vault.Client

	// Missing secret id
	client = Setup("https://google.com", "dead-beef", "", "", "/foo/bar", "example.groovy", "")
	assert.NotNil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return error for empty secret id")

	// Missing role id
	client = Setup("https://google.com", "", "ea7-beef", "", "/foo/bar", "example.groovy", "")
	assert.NotNil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return error for empty role id")

	// Missing path
	client = Setup("https://google.com", "dead-beef", "ea7-beef", "", "", "example.groovy", "")
	assert.NotNil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return error for empty path")

	// Missing file
	client = Setup("https://google.com", "dead-beef", "ea7-beef", "", "/foo/bar", "", "")
	assert.NotNil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return error for empty file")

	// Invalid file
	client = Setup("https://google.com", "dead-beef", "ea7-beef", "", "/foo/bar", "foobar.groovy", "")
	assert.NotNil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return error for invalid file 'foobar.groovy'")

	// Valid role id, secret id, path, and file
	client = Setup("https://google.com", "dead-beef", "ea7-beef", "", "/foo/bar", "example.groovy", "")
	assert.Nil(t, client.ValidateParseFile(), "Expected ValidateParseFile() to return nil for valid role id, secret id, path, and file: %v", client.ValidateParseFile())
}
