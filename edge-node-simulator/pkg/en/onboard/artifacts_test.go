// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard_test

import (
	"testing"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/onboard"
	"github.com/stretchr/testify/assert"
)

var (
	// Test data for the test cases
	urlFilesRS            = "registry-rs.edgeorchestration.intel.com"
	tinkerActionsVersion  = "1.17.1"
	agentsManifestVersion = "1.0.0"
	orchFQDN              = "kind.internal"
	tiverOsVersion        = "1.0.0"
)

func TestGetArtifacts(t *testing.T) {
	cfg := &defs.Settings{
		OrchFQDN:              orchFQDN,
		BaseFolder:            "/tmp",
		EnableDownloads:       true,
		URLFilesRS:            urlFilesRS,
		TinkerActionsVersion:  tinkerActionsVersion,
		AgentsManifestVersion: agentsManifestVersion,
		TiberOSVersion:        tiverOsVersion,
	}

	err := onboard.GetArtifacts(cfg)
	assert.NoError(t, err)
}
