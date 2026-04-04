package types

import "time"

type ArtifactRecord struct {
	ID                 string    `json:"id" yaml:"id"`
	Scope              string    `json:"scope" yaml:"scope"`
	Service            string    `json:"service" yaml:"service"`
	Runtime            string    `json:"runtime" yaml:"runtime"`
	BundleDir          string    `json:"bundleDir" yaml:"bundleDir"`
	Path               string    `json:"path" yaml:"path"`
	ChecksumSHA256     string    `json:"checksumSha256" yaml:"checksumSha256"`
	SizeBytes          int64     `json:"sizeBytes" yaml:"sizeBytes"`
	CreatedAt          time.Time `json:"createdAt" yaml:"createdAt"`
	IntegrityValidated bool      `json:"integrityValidated" yaml:"integrityValidated"`
	RestoreTested      bool      `json:"restoreTested" yaml:"restoreTested"`
	Degraded           bool      `json:"degraded" yaml:"degraded"`
}

type ArtifactSummary struct {
	Count             int   `json:"count" yaml:"count"`
	TotalSizeBytes    int64 `json:"totalSizeBytes" yaml:"totalSizeBytes"`
	IntegrityReady    int   `json:"integrityReady" yaml:"integrityReady"`
	RestoreTested     int   `json:"restoreTested" yaml:"restoreTested"`
	DegradedArtifacts int   `json:"degradedArtifacts" yaml:"degradedArtifacts"`
}

type ProtectedScope struct {
	Name             string    `json:"name" yaml:"name"`
	Runtime          string    `json:"runtime" yaml:"runtime"`
	StatefulServices int       `json:"statefulServices" yaml:"statefulServices"`
	DetectedAt       time.Time `json:"detectedAt" yaml:"detectedAt"`
}

type SchedulerJobStatus struct {
	Name          string    `json:"name" yaml:"name"`
	Cadence       string    `json:"cadence" yaml:"cadence"`
	LastRunAt     time.Time `json:"lastRunAt" yaml:"lastRunAt"`
	LastSuccessAt time.Time `json:"lastSuccessAt" yaml:"lastSuccessAt"`
	LastError     string    `json:"lastError" yaml:"lastError"`
	Enabled       bool      `json:"enabled" yaml:"enabled"`
}
