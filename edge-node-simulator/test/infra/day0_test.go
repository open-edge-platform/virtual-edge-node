// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration tests", Label(e2eLabel), func() {
	var ensimClient ensim.Client
	var infraAPIClient *api.ClientWithResponses
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

		infraAPIClient, err = GetInfraAPIClient(ctx, cfg)
		Expect(err).To(BeNil())

		ensimClient, err = GetENSimClient(ctx, cfg)
		Expect(err).To(BeNil())

		if cfg.Cleanup {
			errCleanup := utils_test.HelperCleanupHosts(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedules(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocations(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("day0 case 01 - Interactive Onboarding (IO)", Label(day0Label), func() {
		It("should do IO by create/check/delete/verify edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("creating edge nodes in Infrastructure Manager simulator")
			enCredentals := &ensimapi.NodeCredentials{
				Project:         cfg.Project,
				OnboardUsername: cfg.EdgeOnboardUser,
				OnboardPassword: cfg.EdgeOnboardPass,
				ApiUsername:     cfg.EdgeAPIUser,
				ApiPassword:     cfg.EdgeAPIPass,
			}
			for _, enUUID := range enUUIDs {
				zlog.Info().Msgf("Creating node %v", enUUID)
				errCreate := ensimClient.Create(ctx, enUUID, enCredentals, false, true)
				Expect(errCreate).To(BeNil())
			}

			By("listing edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(cfg.AmountEdgeNodes))

			for _, simNode := range listNodes {
				Expect(simNode.Credentials.Project).To(Equal(cfg.Project))
				Expect(simNode.Credentials.OnboardUsername).To(Equal(cfg.EdgeOnboardUser))
				Expect(simNode.Credentials.OnboardPassword).To(Equal(cfg.EdgeOnboardPass))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}
			time.Sleep(waitUntilHostsRunning)

			By("listing all hosts in running status from Infrastructure Manager REST API")
			err = InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			By("checking all hosts status from Infrastructure Manager REST API")
			for _, enUUID := range enUUIDs {
				utils_test.CheckHostStatus(ctx, GinkgoTB(), infraAPIClient, enUUID)
			}

			By("deleting all edge nodes from Infrastructure Manager simulator")
			err = ensimClient.DeleteNodes(ctx, 0)
			Expect(err).To(BeNil())

			By("checking all edge nodes are deleted in Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(0))

			By("checking no edge nodes exist in Infrastructure Manager REST API")
			err = InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, 0)
			Expect(err).To(BeNil())
		})
	})

	Describe("day0 case 02 - Non Interactive Onboarding (NIO)", Label(day0Label), func() {
		It("should do NIO by create/check/delete/verify edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("registering hosts in Infrastructure Manager REST API")
			err := InfrastructureManagerAPIRegisterHosts(ctx, infraAPIClient, cfg, enUUIDs)
			Expect(err).To(BeNil())

			By("creating edge nodes in Infrastructure Manager simulator")
			enCredentals := &ensimapi.NodeCredentials{
				Project:         cfg.Project,
				OnboardUsername: cfg.EdgeOnboardUser,
				OnboardPassword: cfg.EdgeOnboardPass,
				ApiUsername:     cfg.EdgeAPIUser,
				ApiPassword:     cfg.EdgeAPIPass,
			}
			for _, enUUID := range enUUIDs {
				zlog.Info().Msgf("Creating node %v", enUUID)
				errCreate := ensimClient.Create(ctx, enUUID, enCredentals, true, true)
				Expect(errCreate).To(BeNil())
			}

			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(cfg.AmountEdgeNodes))

			for _, simNode := range listNodes {
				Expect(simNode.Credentials.Project).To(Equal(cfg.Project))
				Expect(simNode.Credentials.OnboardUsername).To(Equal(cfg.EdgeOnboardUser))
				Expect(simNode.Credentials.OnboardPassword).To(Equal(cfg.EdgeOnboardPass))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}
			time.Sleep(waitUntilHostsRunning)

			By("checking all hosts in running status from Infrastructure Manager REST API")
			err = InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			By("checking all hosts status from Infrastructure Manager REST API")
			for _, enUUID := range enUUIDs {
				utils_test.CheckHostStatus(ctx, GinkgoTB(), infraAPIClient, enUUID)
			}

			By("deleting all edge nodes from Infrastructure Manager simulator")
			err = ensimClient.DeleteNodes(ctx, 0)
			Expect(err).To(BeNil())

			By("checking no edge nodes exist in Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(0))

			By("checking no hosts exist in Infrastructure Manager REST API")
			err = InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, 0)
			Expect(err).To(BeNil())
		})
	})
})
