package vault

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"gopkg.in/resty.v1"
	"logger"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"text/template"
)

var (
	TLSHandshakeTimeout   = 10
	ResponseHeaderTimeout = 20
	ExpectContinueTimeout = 10
	KeepAlive             = 3
)

type VaultClient struct {
	Address  string
	RoleId   string
	SecretId string
	Token    string
	Path     string
	File     string
	Selector string
	Insecure bool

	SystemHealth SystemHealth
	Auth         Auth
	Secret       Secret

	client *resty.Client
	ctx    context.Context
}

type VaultClientErrors struct {
	Errors []string `json:"errors"`
}

func (i *VaultClientErrors) Error() string {
	return strings.Join(i.Errors, ", ")
}

func NewVaultClient(ctx context.Context, addr string, insecure bool) *VaultClient {
	vault := new(VaultClient)
	vault.Address = addr
	vault.Insecure = insecure
	vault.ctx = ctx
	vault.Validate()
	return vault
}

func (v *VaultClient) Validate() {
	// Validate the address is correct
	_, err := url.ParseRequestURI(v.Address)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	// Setup the resty client
	v.Setup()

	// Validate that SystemHealth is okay, this vault instance is ready
	v.SystemHealth.Reload(v)
	if v.SystemHealth.Ready() != true {
		if v.SystemHealth.GetInitialized() != true {
			logger.Warnf("Expected vault to be initialized")
		}

		if v.SystemHealth.GetSealed() == true {
			logger.Warnf("Expected vault to be unsealed")
		}

		if v.SystemHealth.GetStandby() == true {
			logger.Warnf("Expected vault to be active node")
		}

		logger.Fatalf("Vault does not appear to be ready to receive requests.")
	}
}

func (v *VaultClient) Setup() {
	resty.SetRetryCount(5)
	resty.SetRetryWaitTime(3 * time.Second)
	resty.SetRetryMaxWaitTime(30 * time.Second)

	v.client = resty.New()
	v.client.SetHeader("Content-Type", "application/json")
	v.client.SetTransport(&http.Transport{
		DialContext: (&net.Dialer{
			KeepAlive: time.Duration(int64(KeepAlive) * time.Second.Nanoseconds()),
		}).DialContext,
		TLSHandshakeTimeout:   time.Duration(int64(TLSHandshakeTimeout) * time.Second.Nanoseconds()),
		ResponseHeaderTimeout: time.Duration(int64(ResponseHeaderTimeout) * time.Second.Nanoseconds()),
		ExpectContinueTimeout: time.Duration(int64(ExpectContinueTimeout) * time.Second.Nanoseconds()),
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: v.Insecure},
	})
	v.client.SetHostURL(fmt.Sprintf("%v/v1", v.Address))
	v.client.AddRetryCondition(resty.RetryConditionFunc(func(r *resty.Response) (bool, error) { return r.StatusCode() == http.StatusBadRequest, nil }))
	v.client.AddRetryCondition(resty.RetryConditionFunc(func(r *resty.Response) (bool, error) { return r.StatusCode() == http.StatusBadGateway, nil }))
	v.client.AddRetryCondition(resty.RetryConditionFunc(func(r *resty.Response) (bool, error) { return r.StatusCode() == http.StatusGatewayTimeout, nil }))
	v.client.AddRetryCondition(resty.RetryConditionFunc(func(r *resty.Response) (bool, error) { return r.StatusCode() == http.StatusInternalServerError, nil }))
	v.client.AddRetryCondition(resty.RetryConditionFunc(func(r *resty.Response) (bool, error) { return r.StatusCode() == http.StatusServiceUnavailable, nil }))
}

func (v *VaultClient) checkResponseForErrors(response *resty.Response, responseError error, validStatusCodes ...int) {
	// Log a debug message with the raw response
	logger.Debugf("Response Body: %s", response.Body())

	// Check to make sure the response error object is nil--if it is not, may indicate a low-level HTTP error
	if responseError != nil {
		logger.Fatalf("Got low-level HTTP error: %v", responseError)
	}

	// Validate the response HTTP status code against validStatusCodes[]
	if contains(response.StatusCode(), validStatusCodes) != true {
		logger.Fatalf("Response %v was not one of %v: %v", response.StatusCode(), validStatusCodes, response.Error())
	}
}

func contains(expected int, items []int) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}

	return false
}

func (v *VaultClient) CreateToken(roleId, secretId string) string {
	v.RoleId = roleId
	v.SecretId = secretId
	v.ValidateCreateToken()

	v.Token = v.Auth.Approle.Login(v).Auth.ClientToken
	return v.Token
}

func (v *VaultClient) ValidateCreateToken() {
	// Make sure role id is non-empty
	if v.RoleId == "" {
		logger.Fatalf("Role ID cannot be empty")
	}

	// Make sure secret id is non-empty
	if v.SecretId == "" {
		logger.Fatalf("Secret ID cannot be empty")
	}
}

func (v *VaultClient) RenewToken(token string) string {
	v.Token = token
	v.ValidateRenewToken()
	return v.Auth.Token.RenewSelf(v).Auth.ClientToken
}

func (v *VaultClient) ValidateRenewToken() {
	// Make sure token is non-empty
	if v.Token == "" {
		logger.Fatalf("Token cannot be empty")
	}
}

func (v *VaultClient) RevokeToken(token string) {
	v.Token = token
	v.ValidateRevokeToken()
	v.Auth.Token.RevokeSelf(v)

	logger.Infof("Token revoked successfully!")
}

func (v *VaultClient) ValidateRevokeToken() {
	// Make sure token is non-empty
	if v.Token == "" {
		logger.Fatalf("Token cannot be empty")
	}
}

func (v *VaultClient) FetchSecret(token, path, selector string) string {
	v.Token = token
	v.Path = path
	v.Selector = selector
	v.ValidateFetchSecret()
	secrets := v.Secret.Get(v).Data
	var parsed bytes.Buffer

	template, err := template.New("secrets").Parse(v.Selector)
	if err != nil {
		logger.Fatalf("Could not parse template selector '%v': %v", v.Selector, err)
	}

	err = template.Execute(&parsed, secrets)
	if err != nil {
		logger.Fatalf("Could not render template selector '%v': %v", v.Selector, err)
	}

	return parsed.String()
}

func (v *VaultClient) ValidateFetchSecret() {
	// Make sure token is non-empty
	if v.Token == "" {
		logger.Fatalf("Token cannot be empty")
	}

	// Make sure path is non-empty
	if v.Path == "" {
		logger.Fatalf("Path cannot be empty")
	}

	// Make sure selector is non-empty
	if v.Selector == "" {
		logger.Fatalf("Selector cannot be empty")
	}
}

func (v *VaultClient) ParseFile(roleId, secretId, path, file string) {
	// Set vars for parsing the file
	v.RoleId = roleId
	v.SecretId = secretId
	v.Path = path
	v.File = file
	v.ValidateParseFile()

	// Create the token
	v.Token = v.Auth.Approle.Login(v).Auth.ClientToken

	// Fetch secret data
	secrets := v.Secret.Get(v).Data

	// Parse the file contents
	template, err := template.ParseFiles(v.File)
	if err != nil {
		logger.Fatalf("Could not parse template file '%v': %v", v.File, err)
	}

	// Create the new file we will write content to
	f, err := os.Create(v.File)
	if err != nil {
		logger.Fatalf("Could not create file '%v': %v", v.File, err)
	}

	// Write parsed file contents to disk
	err = template.Execute(f, secrets)
	if err != nil {
		logger.Fatalf("Could not render parsed template content '%v' to disk: %v", v.File, err)
	}

	// Revoke the token
	v.Auth.Token.RevokeSelf(v)

	logger.Infof("Successfully parsed secrets from %v to file %v and auto-revoked token!", v.Path, v.File)
}

func (v *VaultClient) ValidateParseFile() {
	// Make sure role id is non-empty
	if v.RoleId == "" {
		logger.Fatalf("Role ID cannot be empty")
	}

	// Make sure secret id is non-empty
	if v.SecretId == "" {
		logger.Fatalf("Secret ID cannot be empty")
	}

	// Make sure path is non-empty
	if v.Path == "" {
		logger.Fatalf("Path cannot be empty")
	}

	// Make sure file is non-empty and accessible
	if _, err := os.Stat(v.File); os.IsNotExist(err) {
		logger.Fatalf("The file to parse %v either does not exist or cannot be accessed, cannot continue", v.File)
	}
}