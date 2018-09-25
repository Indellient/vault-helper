package vault

import (
	"fmt"
	"net/http"
)

var (
	SecretLocation = "/secret"
)

type Secret struct {
	Data map[string]interface{} `json:"data"`
	LeaseDuration int `json:"lease_duration"`
	LeaseId string `json:"lease_id"`
	Renewable bool `json:"renewable"`
}

func (i *Secret) Get(v *VaultClient) *Secret {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetHeader("X-Vault-Token", v.Token).SetResult(i).SetError(VaultClientErrors{}).Get(fmt.Sprintf("%v/%v", SecretLocation, v.Path))

	v.checkResponseForErrors(response, err, http.StatusOK)

	return i
}

