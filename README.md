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

## Invocation

You can specify the following environment variables to help mask secret information from the system `vault-helper` is
running on.

`VAULT_ADDR` - Vault URL
`VAULT_INSECURE` - Set to `true` to disable SSL cert checking
`VAULT_ROLE_ID` - The vault approle role id
`VAULT_SECRET_ID` - The vault approle secret id
`VAULT_TOKEN` - The vault token

To avoid conflicts with habitat double-curly-braces replacements in files, use double-parens instead: `((.username))`

See --help for more information and detailed invocation examples.

## Caveats

Below are a list of known caveats with `vault-helper`.  If you find other limitations with it, please update this section.

### Vault Keys with Hyphens
Vault keys can have a hyphen, as long as it's double-quoted.  Due to how the GO template engine works, when specifying
a substitution like: `(( .user-name ))`, that key `user-name` should be double-quoted: `(( ".user-name" ))`

### Secret Replacement

`vault-helper` assumes that all secrets at a given path like `secret/data/jenkins/unstable/admin` are to be parsed on a single
file at a time.  This is in part due to how `vault-helper` parses and re-writes the file to disk, as well as to simplify
management of secrets.

Vault helper supports either kv-v1 or kv-v2 secret stores, make sure to pass the correct `--path` in at invocation time.

A good rule-of-thumb is to make sure you invoke `vault-helper` once on a single file at a given time.  Do not put secrets
at different paths in the same file to be parsed by `vault-helper`.
