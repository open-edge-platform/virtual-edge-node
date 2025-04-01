// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration test", Label(e2eLabel), func() {
	var ensimClient ensim.Client
	var infraAPIClient *api.ClientWithResponses
	var cancel context.CancelFunc
	var ctx context.Context
	var cfg *flags_test.TestConfig
	var enUUIDs []string
	var site *api.Site
	var region *api.Region

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

		enUUIDs = GenerateUUIDs(cfg)
		Expect(enUUIDs).NotTo(BeNil())

		var err error
		ctx, cancel = context.WithCancel(context.Background())

		infraAPIClient, err = GetInfraAPIClient(ctx, cfg)
		Expect(err).To(BeNil())

		if cfg.Cleanup {
			errCleanup := utils_test.HelperCleanupHosts(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedules(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocations(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
		}

		ensimClient, err = GetENSimClient(ctx, cfg)
		Expect(err).To(BeNil())

		err = ENSIMCreateNodes(ctx, cfg, ensimClient, enUUIDs)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := ensimClient.DeleteNodes(ctx, 0)
		Expect(err).To(BeNil())
		cancel()
	})

	Describe("Day2 Case 01 - Maintenance by Hosts", Label(day2Label), func() {
		It("should check if hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			err := InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			By("setting all hosts OS with update sources / installed packages")
			hosts, err := utils_test.ListHosts(ctx, infraAPIClient, nil)
			Expect(err).To(BeNil())
			Expect(hosts).ToNot(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
				utils_test.UpdateHostOS(ctx, GinkgoTB(),
					infraAPIClient,
					hostID,
					utils_test.OSUpdateSources,
					utils_test.OSInstalledPackages,
					utils_test.OSKernelCmd,
				)
			}

			By("setting all hosts into maintenance mode - single schedule")
			utils_test.SetHostsSingleSched(ctx, GinkgoTB(), infraAPIClient, hostIDs)

			By("verifying that all hosts are in maintenance state")
			time.Sleep(waitHostsMaintenance)
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), infraAPIClient, hostIDs)

			// ToDo check the update agent status (i.e., UPDATING) of the edge node(s) in ensim
		})
	})

	Describe("Day2 Case 02 - Maintenance by Site", Label(day2Label), func() {
		It("should check if site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			err := InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			hosts, err := utils_test.ListHosts(ctx, infraAPIClient, nil)
			Expect(err).To(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}
			By("creating a site")
			site1 := utils_test.CreateSite(ctx, GinkgoTB(), infraAPIClient, utils_test.Site1Request)
			site = site1.JSON201
			By("configuring hosts to the site")
			err = utils_test.ConfigureHostsbyID(ctx, infraAPIClient, hostIDs, *site1.JSON201.ResourceId)
			Expect(err).To(BeNil())
			By("setting site into maintenance mode - single schedule")
			utils_test.SetSitesSingleSched(ctx, GinkgoTB(), infraAPIClient, []string{*site1.JSON201.ResourceId})

			time.Sleep(waitHostsMaintenance)
			By("checking if site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), infraAPIClient, *site.ResourceId)

			By("checking if all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), infraAPIClient, hostIDs)

			By("setting hosts to be unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, infraAPIClient)
			Expect(err).To(BeNil())
		})
	})

	Describe("Day2 Case 03 - Maintenance by Region", Label(day2Label), func() {
		It("should check if region/site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			err := InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			hosts, err := utils_test.ListHosts(ctx, infraAPIClient, nil)
			Expect(err).To(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}
			By("creating a region")
			region1 := utils_test.CreateRegion(ctx, GinkgoTB(), infraAPIClient, utils_test.Region1Request)
			region = region1.JSON201
			siteReq := utils_test.Site1Request
			siteReq.RegionId = region.ResourceId

			By("creating a site and assigning it to the region")
			site1 := utils_test.CreateSite(ctx, GinkgoTB(), infraAPIClient, siteReq)
			site = site1.JSON201
			err = utils_test.ConfigureHostsbyID(ctx, infraAPIClient, hostIDs, *site1.JSON201.ResourceId)
			Expect(err).To(BeNil())

			By("setting the region into maintenance state - single schedule")
			utils_test.SetRegionSingleSched(ctx, GinkgoTB(), infraAPIClient, []string{*region1.JSON201.ResourceId})

			time.Sleep(waitHostsMaintenance)
			By("checking the region is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), infraAPIClient, *region.ResourceId)

			By("checkint the site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), infraAPIClient, *site.ResourceId)

			By("checking all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), infraAPIClient, hostIDs)

			By("checking all hosts are unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, infraAPIClient)
			Expect(err).To(BeNil())
		})
	})

	Describe("Day2 Case 04 - Maintenance by Multiple Region", Label(day2Label), func() {
		It("should check if region/region/site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			err := InfraAPICheckHosts(ctx, infraAPIClient, &filterRunning, cfg.AmountEdgeNodes)
			Expect(err).To(BeNil())

			hosts, err := utils_test.ListHosts(ctx, infraAPIClient, nil)
			Expect(err).To(BeNil())

			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}

			By("creating a root region 1")
			region1 := utils_test.CreateRegion(ctx, GinkgoTB(), infraAPIClient, utils_test.Region1Request)

			By("creating a root region 2")
			utils_test.Region2Request.ParentId = region1.JSON201.ResourceId
			region2 := utils_test.CreateRegion(ctx, GinkgoTB(), infraAPIClient, utils_test.Region2Request)
			utils_test.Region2Request.ParentId = nil

			siteReq := utils_test.Site1Request
			siteReq.RegionId = region2.JSON201.ResourceId

			By("creating a site and assigning it to the region")
			site1 := utils_test.CreateSite(ctx, GinkgoTB(), infraAPIClient, siteReq)
			site = site1.JSON201
			err = utils_test.ConfigureHostsbyID(ctx, infraAPIClient, hostIDs, *site1.JSON201.ResourceId)
			Expect(err).To(BeNil())

			By("setting the region into maintenance state - single schedule")
			utils_test.SetRegionSingleSched(ctx, GinkgoTB(), infraAPIClient, []string{*region1.JSON201.ResourceId})

			time.Sleep(waitHostsMaintenance)
			By("checking the root region 1 is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), infraAPIClient, *region1.JSON201.ResourceId)

			By("checking the region 2 is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), infraAPIClient, *region2.JSON201.ResourceId)

			By("checkint the site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), infraAPIClient, *site.ResourceId)

			By("checking all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), infraAPIClient, hostIDs)

			By("checking all hosts are unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, infraAPIClient)
			Expect(err).To(BeNil())
		})
	})
})
