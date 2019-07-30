# # encoding: utf-8

# The Inspec reference, with examples and extensive documentation, can be
# found at http://inspec.io/docs/reference/resources/
title 'Vault Helper'

control 'vault-helper-1.0' do
  impact 0.7
  title 'Verify Vault Helper'

  # Verify the vault-helper package was installed (could be any origin)
  describe command('find /hab/pkgs -type d -name "vault-helper" | wc -l') do
    its('exit_status') { should eq 0 }
    its('stderr') { should be_empty }
    its('stdout') { should match /^1$/ }
  end

  # Fetch the root token from habitat ring.  Note that this requires the http gateway to NOT have authentication on it
  vault_token = command(%q{curl --silent -X GET http://localhost:9631/census | jq -r '.census_groups | .["vault.default"] | .service_config | .value | .config | .token'}).stdout.strip
  vault_addr = 'http://vault:8200'

  # Invoke vault-helper
  describe command(%Q{VAULT_TOKEN="#{vault_token}" VAULT_ADDR="#{vault_addr}" vault-helper secret --path="vault-helper/credentials" --selector="((.username)) ((.password))"}) do
    its('exit_status') { should eq 0 }
    its('stderr') { should be_empty }
    its('stdout') { should match /^kevin bacon$/ }
  end
end
