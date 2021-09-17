package vault

import "net/http"

var (
	AuthTokenRenewSelfLocation  = "/auth/token/renew-self"
	AuthTokenRevokeSelfLocation = "/auth/token/revoke-self"
)

type Token struct {
	*Response
}

func (i *Token) RenewSelf(v *Client) *Token {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetHeader("X-Vault-Token", v.Token).SetResult(i).SetError(VaultClientErrors{}).Post(AuthTokenRenewSelfLocation)

	v.checkResponseForErrors(response, err, http.StatusOK)

	return i
}

func (i *Token) RevokeSelf(v *Client) {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetHeader("X-Vault-Token", v.Token).SetError(VaultClientErrors{}).Post(AuthTokenRevokeSelfLocation)

	v.checkResponseForErrors(response, err, http.StatusNoContent)
}
