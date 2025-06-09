// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ensim_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var zlog = logging.GetLogger("ensim_test")

func TestENSim_NIO(t *testing.T) {
	zlog.Info().Msg("TestENSim_Case02_NIO Started")

	cfg := flags_test.GetConfig()
	require.NotNil(t, cfg)

	// Set the environment variables - Project
	t.Setenv(utils.ProjectIDEnvVar, cfg.Project)

	certCA, err := utils_test.LoadFile(cfg.CAPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.OrchFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
	require.NoError(t, err)

	httpClient, err := utils_test.GetClientWithCA(certCA)
	require.NoError(t, err)

	err = utils_test.HelperCleanupHostsAPI(ctx, httpClient, cfg)
	require.NoError(t, err)

	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	require.NoError(t, err)
	defer simClient.Close()

	// Test 0 nodes in Infrastructure Manager API and ENSIM
	listNodes, err := simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(listNodes))

	filter := fmt.Sprintf(`%s = %q`, "host_status", "Running")

	totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filter)
	require.NoError(t, err)
	assert.Equal(t, 0, totalHosts)

	// Host UUID
	hostUUID := uuid.New()
	enUUID := hostUUID.String()

	// Create 1 node in Infrastructure Manager SIM
	enCredentals := &ensimapi.NodeCredentials{
		Project:         cfg.Project,
		OnboardUsername: cfg.EdgeOnboardUser,
		OnboardPassword: cfg.EdgeOnboardPass,
		ApiUsername:     cfg.EdgeAPIUser,
		ApiPassword:     cfg.EdgeAPIPass,
	}

	err = simClient.Create(ctx, enUUID, enCredentals, true)
	require.NoError(t, err)

	time.Sleep(time.Second * 5)

	// Test 1 node in Infrastructure Manager API and ENSIM
	listNodes, err = simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(listNodes))

	totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filter)
	require.NoError(t, err)
	assert.Equal(t, 1, totalHosts)

	// Get node from ENSIM and validate it
	simNode, err := simClient.Get(ctx, enUUID)
	require.NoError(t, err)
	assert.NotNil(t, simNode)

	assert.Equal(t, enUUID, simNode.Uuid)
	assert.Equal(t, cfg.Project, simNode.Credentials.Project)
	assert.Equal(t, cfg.EdgeOnboardUser, simNode.Credentials.OnboardUsername)
	assert.Equal(t, cfg.EdgeOnboardPass, simNode.Credentials.OnboardPassword)
	for _, state := range simNode.AgentsStates {
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.CurrentState)
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.DesiredState)
	}

	for _, status := range simNode.Status {
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, status.Mode, "Status mode is not OK for %v", status.Source)
	}

	// Update node in ENSIM and validate change
	enStates := map[ensimapi.AgentType]ensimapi.AgentState{
		ensimapi.AgentType_AGENT_TYPE_TELEMETRY: ensimapi.AgentState_AGENT_STATE_OFF,
		ensimapi.AgentType_AGENT_TYPE_NODE:      ensimapi.AgentState_AGENT_STATE_OFF,
		ensimapi.AgentType_AGENT_TYPE_HD:        ensimapi.AgentState_AGENT_STATE_OFF,
		ensimapi.AgentType_AGENT_TYPE_UPDATE:    ensimapi.AgentState_AGENT_STATE_OFF,
	}
	err = simClient.Update(ctx, enUUID, enStates)
	require.NoError(t, err)

	simNode, err = simClient.Get(ctx, enUUID)
	require.NoError(t, err)
	assert.NotNil(t, simNode)

	assert.Equal(t, enUUID, simNode.Uuid)
	assert.Equal(t, cfg.Project, simNode.Credentials.Project)
	for _, state := range simNode.AgentsStates {
		assert.Equal(t, ensimapi.AgentState_AGENT_STATE_OFF, state.CurrentState)
		assert.Equal(t, ensimapi.AgentState_AGENT_STATE_OFF, state.DesiredState)
	}

	enStates = map[ensimapi.AgentType]ensimapi.AgentState{
		ensimapi.AgentType_AGENT_TYPE_TELEMETRY: ensimapi.AgentState_AGENT_STATE_ON,
		ensimapi.AgentType_AGENT_TYPE_NODE:      ensimapi.AgentState_AGENT_STATE_ON,
		ensimapi.AgentType_AGENT_TYPE_HD:        ensimapi.AgentState_AGENT_STATE_ON,
		ensimapi.AgentType_AGENT_TYPE_UPDATE:    ensimapi.AgentState_AGENT_STATE_ON,
	}
	err = simClient.Update(ctx, enUUID, enStates)
	require.NoError(t, err)

	simNode, err = simClient.Get(ctx, enUUID)
	require.NoError(t, err)
	assert.NotNil(t, simNode)

	assert.Equal(t, enUUID, simNode.Uuid)
	assert.Equal(t, cfg.Project, simNode.Credentials.Project)
	for _, state := range simNode.AgentsStates {
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.CurrentState)
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.DesiredState)
	}

	// Delete 1 node in ENSIM
	err = simClient.Delete(ctx, enUUID)
	require.NoError(t, err)

	// . Test 0 nodes in Infrastructure Manager API and ENSIM
	listNodes, err = simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(listNodes))

	totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filter)
	require.NoError(t, err)
	assert.Equal(t, 0, totalHosts)

	zlog.Info().Msg("TestENSim_Case02_NIO Finished")
}
