// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package defs

var (
	// Default values of configuration parameters.
	OrchUser             = "" // update <orch-user>
	OrchPasswd           = "" // update <orch-pass>
	OrchKcClientID       = "system-client"
	OrchProject          = "sample-project"
	OrchOrg              = "intel"
	DefaultCAPath        = "/usr/local/share/ca-certificates/orch-ca.crt"
	DefaultoamServerAddr = "0.0.0.0:2379"

	DefaultFQDN       = "kind.internal"
	DefaultBaseFolder = "/etc/intel_edge_node"
)

//nolint:gosec // These are not credentials
var (
	ENClientFolder     = "/client-credentials"
	ENClientIDPath     = "/client-credentials/client_id"
	ENClientSecretPath = "/client-credentials/client_secret"
	ENClientTokenPath  = "/client-credentials/access_token"
	ENClientNamePath   = "/client-credentials/client_name"
	ENTenantIDPath     = "/tenantId"
)

var TokenFolders = []string{
	"/tokens/node-agent",
	"/tokens/hd-agent",
	"/tokens/cluster-agent",
	"/tokens/platform-update-agent",
	"/tokens/platform-observability-agent",
	"/tokens/platform-telemetry-agent",
	"/tokens/prometheus",
	"/tokens/license-agent",
}

//nolint:gosec // These are not credentials
var (
	NodeAgentTokenPath      = "/tokens/node-agent/access_token"
	UpdateAgentTokenPath    = "/tokens/platform-update-agent/access_token"
	HDAgentTokenPath        = "/tokens/hd-agent/access_token"
	TelemetryAgentTokenPath = "/tokens/platform-telemetry-agent/access_token"
	LicenseAgentTokenPath   = "/tokens/license-agent/access_token"
)

type Settings struct {
	CertCAPath            string
	CertCA                string
	OrchFQDN              string
	ENGUID                string
	ENSerial              string
	EdgeAPIUser           string
	EdgeAPIPass           string
	EdgeOnboardUser       string
	EdgeOnboardPass       string
	RunAgents             bool
	NIOnboard             bool
	SouthOnboard          bool
	SetupTeardown         bool
	OamServerAddr         string
	BaseFolder            string
	AutoProvision         bool
	Project               string
	Org                   string
	MACAddress            string
	ENiC                  bool
	EnableDownloads       bool
	URLFilesRS            string
	TinkerActionsVersion  string
	AgentsManifestVersion string
}
