package types

import "time"

type BuildInfo struct {
	Name    string
	Version string
	Commit  string
	Date    string
}

type RuntimeTarget string

const (
	RuntimeCompose    RuntimeTarget = "compose"
	RuntimeKubernetes RuntimeTarget = "kubernetes"
)

type SystemStatus struct {
	ServiceName     string          `json:"serviceName" yaml:"serviceName"`
	Version         string          `json:"version" yaml:"version"`
	Mode            string          `json:"mode" yaml:"mode"`
	ConfigSource    string          `json:"configSource" yaml:"configSource"`
	StartedAt       time.Time       `json:"startedAt" yaml:"startedAt"`
	RuntimeTargets  []RuntimeTarget `json:"runtimeTargets" yaml:"runtimeTargets"`
	ProtectedScopes int             `json:"protectedScopes" yaml:"protectedScopes"`
	Artifacts       ArtifactSummary `json:"artifacts" yaml:"artifacts"`
}

type ProtectionState struct {
	RecoveryPointExists bool      `json:"recoveryPointExists" yaml:"recoveryPointExists"`
	IntegrityValidated  bool      `json:"integrityValidated" yaml:"integrityValidated"`
	RestoreTested       bool      `json:"restoreTested" yaml:"restoreTested"`
	Degraded            bool      `json:"degraded" yaml:"degraded"`
	LastValidatedAt     time.Time `json:"lastValidatedAt" yaml:"lastValidatedAt"`
}

type PolicyDecision struct {
	Allow    bool   `json:"allow" yaml:"allow"`
	Severity string `json:"severity" yaml:"severity"`
	Reason   string `json:"reason" yaml:"reason"`
}
