// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ensim_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var zlog = logging.GetLogger("ensim_test")

func TestENSim_Case01_IO(t *testing.T) {
	zlog.Info().Msg("TestENSim_Case01_IO Started")

	cfg := flags_test.GetConfig()
	require.NotNil(t, cfg)

	// Set the environment variables - projectID
	t.Setenv(utils.ProjectIDEnvVar, cfg.ProjectID)

	certCA, err := utils_test.LoadFile(cfg.CAPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.OrchFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
	require.NoError(t, err)

	httpClient, err := utils_test.GetClientWithCA(certCA)
	require.NoError(t, err)

	apiClient, err := api.NewClientWithResponses(cfg.InfraRESTAPIAddress, api.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = utils_test.HelperCleanupHosts(ctx, apiClient)
	require.NoError(t, err)

	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	require.NoError(t, err)
	defer simClient.Close()

	// 1. Test 0 nodes in Infrastructure Manager API and ENSIM
	listNodes, err := simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(listNodes))

	filter := fmt.Sprintf(`%s = %q`, "host_status", "Running")
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 0, *resList.JSON200.TotalElements)

	// 2. Create 1 node in Infrastructure Manager SIM
	enUUID := uuid.New().String()
	enCredentals := &ensimapi.NodeCredentials{
		ProjectId:       cfg.ProjectID,
		OnboardUsername: cfg.EdgeOnboardUser,
		OnboardPassword: cfg.EdgeOnboardPass,
		ApiUsername:     cfg.EdgeAPIUser,
		ApiPassword:     cfg.EdgeAPIPass,
	}
	err = simClient.Create(ctx, enUUID, enCredentals, false, true)
	require.NoError(t, err)

	time.Sleep(time.Second * 5)

	// 3. Test 1 node in Infrastructure Manager API and ENSIM
	listNodes, err = simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(listNodes))

	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 1, *resList.JSON200.TotalElements)

	// . Get node from ENSIM and validate it
	simNode, err := simClient.Get(ctx, enUUID)
	require.NoError(t, err)
	assert.NotNil(t, simNode)

	assert.Equal(t, enUUID, simNode.Uuid)
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
	assert.Equal(t, cfg.EdgeOnboardUser, simNode.Credentials.OnboardUsername)
	assert.Equal(t, cfg.EdgeOnboardPass, simNode.Credentials.OnboardPassword)
	for _, state := range simNode.AgentsStates {
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.CurrentState)
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, state.DesiredState)
	}

	for _, status := range simNode.Status {
		assert.Equal(t, ensimapi.StatusMode_STATUS_MODE_OK, status.Mode, "Status mode is not OK for %v", status.Source)
	}

	// . Update node in ENSIM and validate change
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
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
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
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
	for _, state := range simNode.AgentsStates {
		assert.Equal(t, ensimapi.AgentState_AGENT_STATE_ON, state.CurrentState)
		assert.Equal(t, ensimapi.AgentState_AGENT_STATE_ON, state.DesiredState)
	}

	// . Delete 1 node in ENSIM
	err = simClient.Delete(ctx, enUUID)
	require.NoError(t, err)

	// . Test 0 nodes in Infrastructure Manager API and ENSIM
	listNodes, err = simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(listNodes))

	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 0, *resList.JSON200.TotalElements)

	zlog.Info().Msg("TestENSim_Case01_IO Finished")
}

func TestENSim_Case02_NIO(t *testing.T) {
	zlog.Info().Msg("TestENSim_Case02_NIO Started")

	cfg := flags_test.GetConfig()
	require.NotNil(t, cfg)

	// Set the environment variables - projectID
	t.Setenv(utils.ProjectIDEnvVar, cfg.ProjectID)

	certCA, err := utils_test.LoadFile(cfg.CAPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.OrchFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
	require.NoError(t, err)

	httpClient, err := utils_test.GetClientWithCA(certCA)
	require.NoError(t, err)

	apiClient, err := api.NewClientWithResponses(cfg.InfraRESTAPIAddress, api.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = utils_test.HelperCleanupHosts(ctx, apiClient)
	require.NoError(t, err)

	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	require.NoError(t, err)
	defer simClient.Close()

	// Test 0 nodes in Infrastructure Manager API and ENSIM
	listNodes, err := simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(listNodes))

	filter := fmt.Sprintf(`%s = %q`, "host_status", "Running")
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 0, *resList.JSON200.TotalElements)

	// Host UUID
	hostUUID := uuid.New()
	enUUID := hostUUID.String()
	enSerial := enUUID[:5]
	enName := "TestHost"

	// Register Host
	AutoOnboardTrue := true
	hostRegisterReq := api.HostRegisterInfo{
		Name:         &enName,
		Uuid:         &hostUUID,
		SerialNumber: &enSerial,
		AutoOnboard:  &AutoOnboardTrue,
	}
	hostRegister, err := apiClient.PostComputeHostsRegisterWithResponse(
		ctx,
		hostRegisterReq,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, hostRegister.StatusCode())

	// Create 1 node in Infrastructure Manager SIM
	enCredentals := &ensimapi.NodeCredentials{
		ProjectId:       cfg.ProjectID,
		OnboardUsername: cfg.EdgeOnboardUser,
		OnboardPassword: cfg.EdgeOnboardPass,
		ApiUsername:     cfg.EdgeAPIUser,
		ApiPassword:     cfg.EdgeAPIPass,
	}

	err = simClient.Create(ctx, enUUID, enCredentals, true, true)
	require.NoError(t, err)

	time.Sleep(time.Second * 5)

	// Test 1 node in Infrastructure Manager API and ENSIM
	listNodes, err = simClient.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(listNodes))

	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 1, *resList.JSON200.TotalElements)

	// Get node from ENSIM and validate it
	simNode, err := simClient.Get(ctx, enUUID)
	require.NoError(t, err)
	assert.NotNil(t, simNode)

	assert.Equal(t, enUUID, simNode.Uuid)
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
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
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
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
	assert.Equal(t, cfg.ProjectID, simNode.Credentials.ProjectId)
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

	resList, err = apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: &filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resList.StatusCode())
	require.Equal(t, 0, *resList.JSON200.TotalElements)

	zlog.Info().Msg("TestENSim_Case02_NIO Finished")
}
