package vault

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"gopkg.in/resty.v1"
	"net"
	"net/http"
	"net/url"
	"os"
	path "path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Indellient/vault-helper/pkg/logger"
)

var (
	TLSHandshakeTimeout   = 10
	ResponseHeaderTimeout = 20
	ExpectContinueTimeout = 10
	KeepAlive             = 3
	LeftTemplateDelim     = `((`
	RightTemplateDelim    = `))`
)

// A client represents a go-resty based HTTP client that interacts with the vault API
type Client struct {
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

// When vault emits errors, we marshal them to this struct so it's easier to print out
type VaultClientErrors struct {
	Errors []string `json:"errors"`
}

func (i *VaultClientErrors) Error() string {
	return strings.Join(i.Errors, ", ")
}

// Basic validation of the vault inputs for the URL
func (v *Client) Validate() error {
	// Validate the address is correct
	_, err := url.ParseRequestURI(v.Address)
	if err != nil {
		return err
	}

	return nil
}

// Extended validate is broken out separately here since it makes HTTP calls to vault
// Note that we expect vault to be initialized, unsealed, and the active node to continue.
func (v *Client) ExtendedValidate() error {
	// Setup the resty client
	v.Setup()

	// Validate that SystemHealth is okay, this vault instance is ready
	v.SystemHealth.Reload(v)
	if ! v.SystemHealth.Ready() {
		if ! v.SystemHealth.GetInitialized() {
			return errors.New("Expected vault to be initialized")
		}

		if v.SystemHealth.GetSealed() {
			return errors.New("Expected vault to be unsealed")
		}

		if v.SystemHealth.GetStandby() {
			return errors.New("Expected vault to be active node")
		}

		return errors.New("Vault does not appear to be ready to receive requests.")
	}

	return nil
}

// Sets up the go-resty client to interact with the vault API service. We do set some defaults for retry count/wait/max,
// and our own custom HTTP.Transport so we can ignore self-signed SSL certs if required. We also add a few retry conditions
// if vault is having issues or over-loaded.
func (v *Client) Setup() {
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

// Once we make a request to the vault HTTP API, we always need to verify the response we recieved back from the server
// is what we expected. This also can catch low-level HTTP responses as well (timeout, eof, connection refused) directly
// on the responseError object.
func (v *Client) checkResponseForErrors(response *resty.Response, responseError error, validStatusCodes ...int) {
	// Log a debug message with the raw response
	logger.Debugf("Response Body: %s", response.Body())

	// Check to make sure the response error object is nil--if it is not, may indicate a low-level HTTP error
	if responseError != nil {
		logger.Fatalf("Got low-level HTTP error: %v", responseError)
	}

	// Validate the response HTTP status code against validStatusCodes[]
	if ! v.contains(response.StatusCode(), validStatusCodes) {
		logger.Fatalf("Response %v was not one of %v: %v", response.StatusCode(), validStatusCodes, response.Error())
	}
}

// Silly struct method to determine if expected is contained in items.
func (v *Client) contains(expected int, items []int) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}

	return false
}

// Given the role id and secret id,
func (v *Client) CreateToken(roleId, secretId string) string {
	v.RoleId = roleId
	v.SecretId = secretId

	err := v.ValidateCreateToken()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	v.Token = v.Auth.Approle.Login(v).Auth.ClientToken
	return v.Token
}

func (v *Client) ValidateCreateToken() error {
	// Make sure role id is non-empty
	if v.RoleId == "" {
		return errors.New("Role ID cannot be empty")
	}

	// Make sure secret id is non-empty
	if v.SecretId == "" {
		return errors.New("Secret ID cannot be empty")
	}

	return nil
}

func (v *Client) RenewToken(token string) string {
	v.Token = token

	err := v.ValidateRenewToken()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	return v.Auth.Token.RenewSelf(v).Auth.ClientToken
}

func (v *Client) ValidateRenewToken() error {
	// Make sure token is non-empty
	if v.Token == "" {
		return errors.New("Token cannot be empty")
	}

	return nil
}

func (v *Client) RevokeToken(token string) {
	v.Token = token

	err := v.ValidateRevokeToken()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	v.Auth.Token.RevokeSelf(v)

	logger.Infof("Token revoked successfully!")
}

func (v *Client) ValidateRevokeToken() error {
	// Make sure token is non-empty
	if v.Token == "" {
		return errors.New("Token cannot be empty")
	}

	return nil
}

func (v *Client) FetchSecret(token, path, selector string) string {
	v.Token = token
	v.Path = path
	v.Selector = selector

	err := v.ValidateFetchSecret()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	secrets := v.Secret.Get(v).Data
	var parsed bytes.Buffer

	template, err := template.New("secrets").Delims(LeftTemplateDelim, RightTemplateDelim).Parse(v.Selector)
	if err != nil {
		logger.Fatalf("Could not parse template selector '%v': %v", v.Selector, err)
	}

	err = template.Execute(&parsed, secrets)
	if err != nil {
		logger.Fatalf("Could not render template selector '%v': %v", v.Selector, err)
	}

	return parsed.String()
}

func (v *Client) ValidateFetchSecret() error {
	// Make sure token is non-empty
	if v.Token == "" {
		return errors.New("Token cannot be empty")
	}

	// Make sure path is non-empty
	if v.Path == "" {
		return errors.New("Path cannot be empty")
	}

	// Make sure selector is non-empty
	if v.Selector == "" {
		return errors.New("Selector cannot be empty")
	}

	// Initialize and attempt to parse the token replacement
	_, err := template.New("secrets").Delims(LeftTemplateDelim, RightTemplateDelim).Parse(v.Selector)
	if err != nil {
		return fmt.Errorf("Could not parse template selector '%v': %v", v.Selector, err)
	}

	return nil
}

func (v *Client) ParseFile(roleId, secretId, vaultPath, file string) {
	// Set vars for parsing the file
	v.RoleId = roleId
	v.SecretId = secretId
	v.Path = vaultPath
	v.File = file

	err := v.ValidateParseFile()
	if err != nil {
		logger.Fatalf("%v", err)
	}

	// Create the token
	v.Token = v.Auth.Approle.Login(v).Auth.ClientToken

	// Fetch secret data
	secrets := v.Secret.Get(v).Data

	// Parse the file contents
	template, err := template.New(path.Base(v.File)).Delims(LeftTemplateDelim, RightTemplateDelim).ParseFiles(v.File)
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

func (v *Client) ValidateParseFile() error {
	// Make sure role id is non-empty
	if v.RoleId == "" {
		return errors.New("Role ID cannot be empty")
	}

	// Make sure secret id is non-empty
	if v.SecretId == "" {
		return errors.New("Secret ID cannot be empty")
	}

	// Make sure path is non-empty
	if v.Path == "" {
		return errors.New("Path cannot be empty")
	}

	// Make sure file is non-empty and accessible
	if _, err := os.Stat(v.File); os.IsNotExist(err) {
		return fmt.Errorf("The file to parse %v either does not exist or cannot be accessed: %v", v.File, err)
	}

	return nil
}
