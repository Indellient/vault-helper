package vault

type Auth struct {
	ClientToken      string            `json:"client_token"`
	Accessor         string            `json:"accessor"`
	Policies         []string          `json:"policies"`
	TokenPolicies    []string          `json:"token_policies,omitempty"`
	IdentityPolicies []string          `json:"identity_policies,omitempty"`
	Metadata         map[string]string `json:"metadata"`
	LeaseDuration    int               `json:"lease_duration"`
	Renewable        bool              `json:"renewable"`
	EntityID         string            `json:"entity_id"`
	Approle          Approle
	Token            Token
}
