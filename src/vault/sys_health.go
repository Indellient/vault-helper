package vault

import (
	"net/http"
)

var (
	SysHealthLocation = "/sys/health"
)

type SystemHealth struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
	Standby     bool `json:"standby"`
}

func (i *SystemHealth) Reload(v *Client) *SystemHealth {
	response, err := v.client.NewRequest().SetContext(v.ctx).SetResult(i).SetError(VaultClientErrors{}).Get(SysHealthLocation)

	v.checkResponseForErrors(response, err, http.StatusOK)

	return i
}

func (i *SystemHealth) Ready() bool {
	return i.GetInitialized() == true && i.GetSealed() == false && i.GetStandby() == false
}

func (i *SystemHealth) GetInitialized() bool {
	return i.Initialized
}

func (i *SystemHealth) GetSealed() bool {
	return i.Sealed
}

func (i *SystemHealth) GetStandby() bool {
	return i.Standby
}
