package iam

// PlatformPermission is one assignable platform permission point (resource/action).
type PlatformPermission struct {
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	Domain     string `json:"domain"`
	Label      string `json:"label"`
	IsHighRisk bool   `json:"is_high_risk"`
}
