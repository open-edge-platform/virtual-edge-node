// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/infra-core/apiv2/v2/pkg/api/v2"
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
	var site *api.SiteResource
	var region *api.RegionResource

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

		if cfg.Cleanup {
			errCleanup := utils_test.HelperCleanupHostsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedulesAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocationsAPI(ctx, httpClient, cfg)
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
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("setting all hosts OS with update sources / installed packages")
			hosts, err := utils_test.ListHosts(ctx, httpClient, nil)
			Expect(err).To(BeNil())
			Expect(hosts).ToNot(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
				utils_test.UpdateHostOS(ctx, GinkgoTB(),
					httpClient,
					hostID,
					utils_test.OSInstalledPackages,
				)
			}

			By("setting all hosts into maintenance mode - single schedule")
			utils_test.SetHostsSingleSched(ctx, GinkgoTB(), httpClient, hostIDs)

			By("verifying that all hosts are in maintenance state")
			time.Sleep(waitHostsMaintenance)
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), httpClient, hostIDs)

			// ToDo check the update agent status (i.e., UPDATING) of the edge node(s) in ensim
		})
	})

	Describe("Day2 Case 02 - Maintenance by Site", Label(day2Label), func() {
		It("should check if site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			hosts, err := utils_test.ListHosts(ctx, httpClient, nil)
			Expect(err).To(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}
			By("creating a site")
			site1ResourceID, err := utils_test.CreateSiteAPI(ctx, httpClient, cfg, &utils_test.Site1Request)
			Expect(err).To(BeNil())

			By("configuring hosts to the site")
			err = utils_test.ConfigureHostsbyID(ctx, httpClient, hostIDs, site1ResourceID)
			Expect(err).To(BeNil())
			By("setting site into maintenance mode - single schedule")
			utils_test.SetSitesSingleSched(ctx, GinkgoTB(), httpClient, []string{site1ResourceID})

			time.Sleep(waitHostsMaintenance)
			By("checking if site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), httpClient, *site.ResourceId)

			By("checking if all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), httpClient, hostIDs)

			By("setting hosts to be unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, httpClient)
			Expect(err).To(BeNil())
		})
	})

	Describe("Day2 Case 03 - Maintenance by Region", Label(day2Label), func() {
		It("should check if region/site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			hosts, err := utils_test.ListHosts(ctx, httpClient, nil)
			Expect(err).To(BeNil())
			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}
			By("creating a region")
			region1ResourceID, err := utils_test.CreateRegionAPI(ctx, httpClient, cfg, &utils_test.Region1Request)
			Expect(err).To(BeNil())
			siteReq := utils_test.Site1Request
			siteReq.RegionId = &region1ResourceID

			By("creating a site and assigning it to the region")
			site1ResourceID, err := utils_test.CreateSiteAPI(ctx, httpClient, cfg, &siteReq)
			Expect(err).To(BeNil())

			err = utils_test.ConfigureHostsbyID(ctx, httpClient, hostIDs, site1ResourceID)
			Expect(err).To(BeNil())

			By("setting the region into maintenance state - single schedule")
			utils_test.SetRegionSingleSched(ctx, GinkgoTB(), httpClient, []string{region1ResourceID})

			time.Sleep(waitHostsMaintenance)
			By("checking the region is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), httpClient, *region.ResourceId)

			By("checkint the site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), httpClient, *site.ResourceId)

			By("checking all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), httpClient, hostIDs)

			By("checking all hosts are unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, httpClient)
			Expect(err).To(BeNil())
		})
	})

	Describe("Day2 Case 04 - Maintenance by Multiple Region", Label(day2Label), func() {
		It("should check if region/region/site/hosts can be scheduled in maintenance state", func(ctx SpecContext) {
			time.Sleep(waitUntilHostsRunning)
			By("listing all hosts in running status")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			hosts, err := utils_test.ListHosts(ctx, httpClient, nil)
			Expect(err).To(BeNil())

			hostIDs := []string{}
			for hostID := range hosts {
				hostIDs = append(hostIDs, hostID)
			}

			By("creating a root region 1")
			region1ResourceID, err := utils_test.CreateRegionAPI(ctx, httpClient, cfg, &utils_test.Region1Request)
			Expect(err).To(BeNil())

			By("creating a root region 2")
			utils_test.Region2Request.ParentId = &region1ResourceID
			region2ResourceID, err := utils_test.CreateRegionAPI(ctx, httpClient, cfg, &utils_test.Region2Request)
			utils_test.Region2Request.ParentId = nil
			Expect(err).To(BeNil())

			siteReq := utils_test.Site1Request
			siteReq.RegionId = &region2ResourceID

			By("creating a site and assigning it to the region")
			site1ResourceID, err := utils_test.CreateSiteAPI(ctx, httpClient, cfg, &siteReq)
			Expect(err).To(BeNil())

			err = utils_test.ConfigureHostsbyID(ctx, httpClient, hostIDs, site1ResourceID)
			Expect(err).To(BeNil())

			By("setting the region into maintenance state - single schedule")
			utils_test.SetRegionSingleSched(ctx, GinkgoTB(), httpClient, []string{region1ResourceID})

			time.Sleep(waitHostsMaintenance)
			By("checking the root region 1 is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), httpClient, region1ResourceID)

			By("checking the region 2 is in maintenance state")
			utils_test.CheckRegionMaintenance(ctx, GinkgoTB(), httpClient, region2ResourceID)

			By("checkint the site is in maintenance state")
			utils_test.CheckSiteMaintenance(ctx, GinkgoTB(), httpClient, *site.ResourceId)

			By("checking all hosts are in maintenance state")
			Expect(err).To(BeNil())
			utils_test.CheckHostsMaintenance(ctx, GinkgoTB(), httpClient, hostIDs)

			By("checking all hosts are unconfigured from site")
			err = utils_test.UnconfigureAllHosts(ctx, httpClient)
			Expect(err).To(BeNil())
		})
	})
})
