package vault

import (
	"net/http"
)

var (
	AuthApproleLoginLocation = "/auth/approle/login"
)

type ApproleLoginInput struct {
	RoleId   string `json:"role_id"`
	SecretId string `json:"secret_id"`
}

type Approle struct {
	*Response
}

func (i *Approle) Login(v *Client) *Approle {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetBody(&ApproleLoginInput{RoleId: v.RoleId, SecretId: v.SecretId}).SetResult(i).SetError(VaultClientErrors{}).Post(AuthApproleLoginLocation)

	v.checkResponseForErrors(response, err, http.StatusOK)

	return i
}
