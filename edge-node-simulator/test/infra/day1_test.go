// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration test", Label(e2eLabel), func() {
	var ensimClient ensim.Client
	var httpClient *http.Client
	var cancel context.CancelFunc
	var ctx context.Context
	var cfg *flags_test.TestConfig
	var enUUIDs []string

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

		enUUIDs = GenerateUUIDs(cfg)
		Expect(enUUIDs).NotTo(BeNil())

		var err error
		ctx, cancel = context.WithCancel(context.Background())

		certCA, err := utils_test.LoadFile(cfg.CAPath)
		Expect(err).To(BeNil())

		httpClient, err = utils_test.GetClientWithCA(certCA)
		Expect(err).To(BeNil())

		ctx, cancel = context.WithCancel(context.Background())
		Expect(ctx).NotTo(BeNil())
		Expect(cancel).NotTo(BeNil())

		err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.ClusterFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
		Expect(err).To(BeNil())

		ensimClient, err = GetENSimClient(ctx, cfg)
		Expect(err).To(BeNil())

		if cfg.Cleanup {
			errCleanup := utils_test.HelperCleanupHostsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedulesAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocationsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
		}

		err = ENSIMCreateNodes(ctx, cfg, ensimClient, enUUIDs)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := ensimClient.DeleteNodes(ctx, 0)
		Expect(err).To(BeNil())
		cancel()
	})

	Describe("Day1 Case 01 - Hosts running in steady state", Label(day1Label), func() {
		It("should wait and check all hosts in running status in Infrastructure Manager REST API"+
			"and Infrastructure Manager simulator", func(ctx SpecContext) {
			By("checking all hosts in running status in Infrastructure Manager REST API")
			time.Sleep(waitHostsRunning)
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("should be able to list edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			for _, simNode := range listNodes {
				Expect(cfg.Project).To(Equal(simNode.Credentials.Project))
				Expect(cfg.EdgeOnboardUser).To(Equal(simNode.Credentials.OnboardUsername))
				Expect(cfg.EdgeOnboardPass).To(Equal(simNode.Credentials.OnboardPassword))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}
		})
	})

	Describe("Day1 Case 02 - Hosts connection lost", Label(day1Label), func() {
		It("should verify connection lost and running status in"+
			"Infrastructure Manager REST API and Infrastructure Manager simulator", func(ctx SpecContext) {
			By("checking all hosts in running status")
			time.Sleep(waitUntilHostsRunning)
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))
			By("should be able to list edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			for _, simNode := range listNodes {
				Expect(cfg.Project).To(Equal(simNode.Credentials.Project))
				Expect(cfg.EdgeOnboardUser).To(Equal(simNode.Credentials.OnboardUsername))
				Expect(cfg.EdgeOnboardPass).To(Equal(simNode.Credentials.OnboardPassword))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}

			By("turning off all agents in all edge nodes")
			// Stops all edge node agents
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			enStates := map[ensimapi.AgentType]ensimapi.AgentState{
				ensimapi.AgentType_AGENT_TYPE_TELEMETRY: ensimapi.AgentState_AGENT_STATE_OFF,
				ensimapi.AgentType_AGENT_TYPE_NODE:      ensimapi.AgentState_AGENT_STATE_OFF,
				ensimapi.AgentType_AGENT_TYPE_HD:        ensimapi.AgentState_AGENT_STATE_OFF,
				ensimapi.AgentType_AGENT_TYPE_UPDATE:    ensimapi.AgentState_AGENT_STATE_OFF,
			}
			for _, simNode := range listNodes {
				// Update node in ENSIM and validate change
				err = ensimClient.Update(ctx, simNode.Uuid, enStates)
				Expect(err).To(BeNil())
			}

			By("verifying all agents in all edge nodes are turned OFF")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			for _, simNode := range listNodes {
				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_OFF))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_OFF))
				}
			}
			Expect(err).To(BeNil())

			By("checking waiting all hosts in no connection status")
			time.Sleep(waitHostsConnectionLost)

			totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterNoConnection)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			totalHosts, err = utils_test.ListInstancesTotalAPI(ctx, httpClient, cfg, &filterInstanceStatusError)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("turning on all agents in all edge nodes")
			// Stops all edge node agents
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			enStates = map[ensimapi.AgentType]ensimapi.AgentState{
				ensimapi.AgentType_AGENT_TYPE_TELEMETRY: ensimapi.AgentState_AGENT_STATE_ON,
				ensimapi.AgentType_AGENT_TYPE_NODE:      ensimapi.AgentState_AGENT_STATE_ON,
				ensimapi.AgentType_AGENT_TYPE_HD:        ensimapi.AgentState_AGENT_STATE_ON,
				ensimapi.AgentType_AGENT_TYPE_UPDATE:    ensimapi.AgentState_AGENT_STATE_ON,
			}
			for _, simNode := range listNodes {
				// Update node in ENSIM and validate change
				err = ensimClient.Update(ctx, simNode.Uuid, enStates)
				Expect(err).To(BeNil())
			}

			By("checking edge nodes state/status ON/OK from Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(cfg.AmountEdgeNodes).To(Equal(len(listNodes)))

			for _, simNode := range listNodes {
				Expect(cfg.Project).To(Equal(simNode.Credentials.Project))
				Expect(cfg.EdgeOnboardUser).To(Equal(simNode.Credentials.OnboardUsername))
				Expect(cfg.EdgeOnboardPass).To(Equal(simNode.Credentials.OnboardPassword))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}

			By("listing all hosts exist in running status")
			time.Sleep(waitHostsRunning)
			totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))
		})
	})
})
