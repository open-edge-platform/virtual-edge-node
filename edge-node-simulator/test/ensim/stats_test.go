// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package ensim_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
)

func TestENSim_Stats(t *testing.T) {
	zlog.Info().Msg("TestENSim_Stats Started")

	cfg := flags_test.GetConfig()
	require.NotNil(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	require.NoError(t, err)
	defer simClient.Close()

	listNodes, err := simClient.List(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, 0, len(listNodes))

	setupFaulty := func(node *ensimapi.Node) ([]*ensimapi.NodeStatus, bool) {
		faults := []*ensimapi.NodeStatus{}
		status := node.GetStatus()
		for _, stat := range status {
			if stat.GetMode() != ensimapi.StatusMode_STATUS_MODE_OK {
				faults = append(faults, stat)
			}
		}
		if len(faults) > 0 {
			return faults, true
		}
		return nil, false
	}

	agentsOff := func(node *ensimapi.Node) ([]string, bool) {
		statesOff := []string{}
		states := node.GetAgentsStates()
		for _, state := range states {
			if state.GetCurrentState() == ensimapi.AgentState_AGENT_STATE_OFF {
				statesOff = append(statesOff, state.GetAgentType().String())
			}
		}

		if len(statesOff) > 0 {
			return statesOff, true
		}
		return nil, false
	}

	faultyNodes := 0
	faultyAgents := 0

	faultsStats := map[string]int{}
	faultsReasons := map[string]int{}
	processFaults := func(faults []*ensimapi.NodeStatus) {
		for _, fault := range faults {
			faultsStats[fault.GetSource().String()]++
			faultsReasons[fault.GetSource().String()+"-"+fault.GetDetails()]++
		}
	}

	agentsOffStats := map[string]int{}
	processAgentsOff := func(faults []string) {
		for _, fault := range faults {
			agentsOffStats[fault]++
		}
	}

	for _, node := range listNodes {
		faults, hasFaults := setupFaulty(node)
		agents, agentsOff := agentsOff(node)

		if hasFaults {
			faultyNodes++
			processFaults(faults)
		}
		if agentsOff {
			faultyAgents++
			processAgentsOff(agents)
		}
	}

	zlog.Info().Msgf("TestENSim_Stats summary: faulty %d nodes, %d agents, total %d", faultyNodes, faultyAgents, len(listNodes))
	zlog.Info().Msgf("Faults stats: %v", faultsStats)
	zlog.Info().Msgf("Faults details (total %d): %v", len(faultsReasons), faultsReasons)
	zlog.Info().Msgf("Agents Off: %v", agentsOffStats)
	zlog.Info().Msg("TestENSim_Stats Finished")
}
