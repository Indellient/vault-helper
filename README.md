# vault-helper

## Summary

This is the vault-helper repo built using golang and Habitat.

## Building

To build the repo, check it out from GitHub, and enter a local studio. Run `build`, the resulting binaries are output
to `bin/vault-helper-*`, and packaged in to the Habitat .hart file.

You can specify `DO_INSTALL=false` if you want a quick `build` command that lets you iterate on the build + test + change 
cycle without Habitat getting in the way.

## Unit Test

The only package that has unit tests right now is the `vault` package, specifically the `Client{}` object. This is 
mostly to cover cases where we may get invalid input from a user.

## Runtime

You can specify the following environment variables to help mask secret information from the system `vault-helper` is
running on.

`VAULT_ADDR` - Vault URL
`VAULT_INSECURE` - Set to `true` to disable SSL cert checking
`VAULT_ROLE_ID` - The vault approle role id
`VAULT_SECRET_ID` - The vault approle secret id
`VAULT_TOKEN` - The vault token

See --help for more information.
