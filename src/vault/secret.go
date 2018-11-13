package vault

import (
	"net/http"
)

type Secret struct {
	Data          map[string]interface{} `json:"data"`
	LeaseDuration int                    `json:"lease_duration"`
	LeaseId       string                 `json:"lease_id"`
	Renewable     bool                   `json:"renewable"`
}

func (i *Secret) Get(v *Client) *Secret {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetHeader("X-Vault-Token", v.Token).SetResult(i).SetError(VaultClientErrors{}).Get(v.Path)

	v.checkResponseForErrors(response, err, http.StatusOK)

	return i
}
