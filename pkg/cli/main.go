package cli

import (
	"context"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	path "path/filepath"
	"strconv"

	"github.com/Indellient/vault-helper/pkg/logger"
	"github.com/Indellient/vault-helper/pkg/vault"
)

const (
	EnvVaultAddr     = "VAULT_ADDR"
	EnvVaultInsecure = "VAULT_SKIP_VERIFY"
	EnvVaultRoleId   = "VAULT_ROLE_ID"
	EnvVaultSecretId = "VAULT_SECRET_ID"
	EnvVaultToken    = "VAULT_TOKEN"
)

var (
	// Build time parameters
	BuildVersion   string
	BuildTimestamp string

	filename = path.Base(os.Args[0])

	app = kingpin.New(filename, fmt.Sprintf(`Description:
	A command-line vault secrets fetcher and template parser.

	When invoking with 'parse', a token is generated, used, and automatically revoked.

	If a token is created or renewed, it must be revoked manually with 'revoke'.

	Vault environment variables VAULT_ADDR, VAULT_SKIP_VERIFY, VAULT_ROLE_ID, VAULT_SECRET_ID, VAULT_TOKEN override command
	line options.

Usage:
	Generate a new approle token:
		%v token create --addr="http://somewhere:8200" --role-id="dead-beef" --secret-id="ea7-beef"

	Renew an existing token (non-zero exit if the token cannot be renewed):
		%v token renew --addr="http://somewhere:8200" --token="dead-c0de"

	Revoke an existing token (non-zero exit if the token cannot be revoked):
		%v token revoke --addr="http://somewhere:8200" --token="dead-c0de"

	Fetch a secret:
		%v secret --addr="http://somewhere:8200" --token="dead-c0de" --path="secret/data/jenkins/dev/user/admin" --selector="((.username))" 
	
	Parse a file:
		%v parse --addr="http://somewhere:8200" --role-id="dead-beef" --secret-id="ea7-beef" --path="secret/data/jenkins/dev/user/admin" --file="init.groovy"
`, filename, filename, filename, filename, filename))

	addr     = app.Flag("addr", "Vault address, like https://somewhere:8200 (VAULT_ADDR)").String()
	insecure = app.Flag("skip-verify", "Skip SSL certificate verification (VAULT_SKIP_VERIFY)").Bool()
	logLevel = app.Flag("log-level", "Logging level, one of: panic, fatal, error, warn, info, debug").Default("error").String()

	token = app.Command("token", "Perform operations on a token")

	// Create a token
	tCreate         = token.Command("create", "Create a new token using the specified role_id and secret_id, printed to STDOUT.")
	tCreateRoleId   = tCreate.Flag("role-id", "The Vault Approle Role Id (VAULT_ROLE_ID)").String()
	tCreateSecretId = tCreate.Flag("secret-id", "The Vault Approle Secret Id (VAULT_SECRET_ID)").String()

	// Renew a token
	tRenew      = token.Command("renew", "Renew an existing token. If it cannot be renewed, command returns non-zero exit status.")
	tRenewToken = tRenew.Flag("token", "The token to be renewed (VAULT_TOKEN).").String()

	// Revoke a token
	tRevoke      = token.Command("revoke", "Revoke an existing token. If it cannot be revoked, command returns non-zero exit status.")
	tRevokeToken = tRevoke.Flag("token", "The token to be revoked (VAULT_TOKEN).").String()

	// Fetch a secret
	secret    = app.Command("secret", "Fetch a given secret from Vault using the specified token, printing to STDOUT.")
	sToken    = secret.Flag("token", "The token used to fetch the secret (VAULT_TOKEN).").String()
	sPath     = secret.Flag("path", "The vault path for the secret, like 'secret/jenkins/dev/user/admin'.").Required().String()
	sSelector = secret.Flag("selector", "The valid go template selector, like '((.username))'.").Required().String()

	// Parse a file
	parse     = app.Command("parse", "Parses all golang template placeholders like '((.username))' in a file, replaced with their secret value from Vault.")
	pRoleId   = parse.Flag("role-id", "The Vault Approle Role Id (VAULT_ROLE_ID)").String()
	pSecretId = parse.Flag("secret-id", "The Vault Approle Secret Id (VAULT_SECRET_ID)").String()
	pPath     = parse.Flag("path", "The vault path for the secret, like 'secret/jenkins/dev/user/admin'.").Required().String()
	pFile     = parse.Flag("file", "The file to perform parsing on.").Required().String()

	// Version
	version = app.Command("version", "Display version and build information")
)

func Run(ctx context.Context, args []string) {
	switch kingpin.MustParse(app.Parse(args[1:])) {
	case tCreate.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Create token ...")
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).CreateToken(GetEnvValue(EnvVaultRoleId, *tCreateRoleId), GetEnvValue(EnvVaultSecretId, *tCreateSecretId)))

	case tRenew.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Renew token ...")
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).RenewToken(GetEnvValue(EnvVaultToken, *tRenewToken)))

	case tRevoke.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Revoke token ...")
		vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).RevokeToken(GetEnvValue(EnvVaultToken, *tRevokeToken))

	case secret.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Fetch secrets from %v ...", *sPath)
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).FetchSecret(GetEnvValue(EnvVaultToken, *sToken), *sPath, *sSelector))

	case parse.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Parse file %v using secrets from %v...", *pFile, *pPath)
		vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).ParseFile(GetEnvValue(EnvVaultRoleId, *pRoleId), GetEnvValue(EnvVaultSecretId, *pSecretId), *pPath, *pFile)

	case version.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		fmt.Println(fmt.Sprintf("%v v%v built on %v", filename, BuildVersion, BuildTimestamp))
	}
}

func GetEnvValue(environmentKey, defaultValue string) string {
	value := os.Getenv(environmentKey)
	if value != "" {
		return value
	}

	return defaultValue
}

func GetBoolEnvValue(environmentKey string, defaultValue bool) bool {
	value := os.Getenv(environmentKey)
	if value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			logger.Fatalf("Could not parse env '%v=%v' to boolean value: %v", environmentKey, value, err)
		}

		return parsed
	}

	return defaultValue
}
