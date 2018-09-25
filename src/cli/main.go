package cli

import (
	"context"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"logger"
	"os"
	"strconv"
	"vault"
)

const (
	EnvVaultAddr     = "VAULT_ADDR"
	EnvVaultInsecure = "VAULT_INSECURE"
	EnvVaultRoleId   = "VAULT_ROLE_ID"
	EnvVaultSecretId = "VAULT_SECRET_ID"
	EnvVaultToken    = "VAULT_TOKEN"
)

var (
	app = kingpin.New(os.Args[0], fmt.Sprintf(`Description:
	A command-line vault secrets fetcher and parser.

	When invoking with 'parse', a token is generated, used, and automatically revoked. If a token is created or renewed,
	it must be revoked manually with 'revoke'.

	Vault environment variables VAULT_ADDR, VAULT_INSECURE, VAULT_ROLE_ID, VAULT_SECRET_ID, VAULT_TOKEN override command
	line specified switches.

Usage:
	Generate a new approle token:
		%v token create --addr="http://somewhere:8200" --role-id="dead-beef" --secret-id="ea7-beef"

	Renew an existing token (non-zero exit if the token cannot be renewed):
		%v token renew --addr="http://somewhere:8200" --token="dead-c0de"

	Revoke an existing token (non-zero exit if the token cannot be revoked):
		%v token revoke --addr="http://somewhere:8200" --token="dead-c0de"

	Fetch a secret:
		%v secret get --addr="http://somewhere:8200" --token="dead-c0de" --path="jenkins/dev/user/admin" --selector="{{.username}}" 
	
	Parse a file:
		%v parse --addr="http://somewhere:8200" --role-id="dead-beef" --secret-id="ea7-beef" --path="jenkins/dev/user/admin" --file="init.groovy"
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0]))

	addr     = app.Flag("addr", "Vault address, like https://somewhere:8200 (VAULT_ADDR)").String()
	insecure = app.Flag("insecure", "Skip SSL certificate verification (VAULT_INSECURE)").Bool()
	logLevel = app.Flag("log-level", "Logging level, like 'error' or 'debug'").Default("error").String()

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
	sSelector = secret.Flag("selector", "The valid go template selector, like '{{.username}}'.").Required().String()

	// Parse a file
	parse     = app.Command("parse", "Parses all golang template placeholders like '{{.username}}' in a file, replaced with their secret value from Vault.")
	pRoleId   = parse.Flag("role-id", "The Vault Approle Role Id (VAULT_ROLE_ID)").String()
	pSecretId = parse.Flag("secret-id", "The Vault Approle Secret Id (VAULT_SECRET_ID)").String()
	pPath     = parse.Flag("path", "The vault path for the secret, like 'jenkins/dev/user/admin'.").Required().String()
	pFile     = parse.Flag("file", "The file to perform parsing on.").Required().String()

	// Build time parameters
	BuildTag       string
	BuildTimestamp string
)

func Run(ctx context.Context, args []string) {
	switch kingpin.MustParse(app.Parse(args[1:])) {
	case tCreate.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Create Token...")
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).CreateToken(GetEnvValue(EnvVaultRoleId, *tCreateRoleId), GetEnvValue(EnvVaultRoleId, *tCreateSecretId)))

	case tRenew.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Renew Token...")
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).RenewToken(GetEnvValue(EnvVaultToken, *tRenewToken)))

	case tRevoke.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Revoke Token...")
		vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).RevokeToken(GetEnvValue(EnvVaultToken, *tRevokeToken))

	case secret.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Fetch Secret...")
		fmt.Println(vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).FetchSecret(GetEnvValue(EnvVaultToken, *sToken), *sPath, *sSelector))

	case parse.FullCommand():
		logger.SetLoggingLevel(*logLevel)
		logger.Infof("Parse File...")
		vault.NewVaultClient(ctx, GetEnvValue(EnvVaultAddr, *addr), GetBoolEnvValue(EnvVaultInsecure, *insecure)).ParseFile(GetEnvValue(EnvVaultRoleId, *pRoleId), GetEnvValue(EnvVaultRoleId, *pSecretId), *pPath, *pFile)
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
