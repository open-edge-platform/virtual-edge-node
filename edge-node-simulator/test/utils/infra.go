// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	host_status "github.com/open-edge-platform/infra-managers/host/pkg/status"
	maint_status "github.com/open-edge-platform/infra-managers/maintenance/pkg/status"
	onb_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	numHostsLargeHierarchy = 10000
	numSitesLargeHierarchy = 1000

	rateLimitInterval = 1 * time.Second
)

var (
	ctxTimeout = time.Second * 20

	requestsBatchSize = 50
	requestsInterval  = 10 * time.Second

	hostUUID        = strings.ToLower("BFD3B398-9A4B-480D-AB53-4050ED108F5C")
	singleSchedName = "schedSingle"
)

func ListHosts(ctx context.Context, apiClient *api.ClientWithResponses, filter *string) (map[string]string, error) {
	zlog.Info().Msg("ListHosts")
	hostsList := make(map[string]string)

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetComputeHostsWithResponse(
			ctx,
			&api.GetComputeHostsParams{
				Offset:   &offset,
				PageSize: &pageSize,
				Filter:   filter,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list hosts")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		for _, host := range *resList.JSON200.Hosts {
			hostsList[*host.ResourceId] = host.Uuid.String()
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All hosts listed %v", hostsList)
			break
		}

		offset += pageSize
	}
	return hostsList, nil
}

// ListInstancesWithFilter returns a map of instances selected by filter and indexed by instanceID associated with hostID.
func ListInstances(ctx context.Context, apiClient *api.ClientWithResponses, filter *string) (map[string]string, error) {
	zlog.Info().Msg("ListInstancesWithFilter")
	instancesList := make(map[string]string)

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetInstancesWithResponse(
			ctx,
			&api.GetInstancesParams{
				Offset:   &offset,
				PageSize: &pageSize,
				Filter:   filter,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list instances")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		if resList.JSON200.Instances != nil {
			for _, inst := range *resList.JSON200.Instances {
				instancesList[*inst.InstanceID] = ""
				if inst.Host != nil && inst.Host.ResourceId != nil {
					instancesList[*inst.InstanceID] = *inst.Host.ResourceId
				}
			}
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All instances listed %v", instancesList)
			break
		}

		offset += pageSize
	}
	return instancesList, nil
}

func DeleteInstance(ctx context.Context, apiClient *api.ClientWithResponses, instanceID string) error {
	resDelInst, err := apiClient.DeleteInstancesInstanceIDWithResponse(
		ctx,
		instanceID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelInst.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func DeleteHost(
	ctx context.Context,
	apiClient *api.ClientWithResponses,
	hostID string,
) error {
	resDelHost, err := apiClient.DeleteComputeHostsHostIDWithResponse(
		ctx,
		hostID,
		api.DeleteComputeHostsHostIDJSONRequestBody{},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelHost.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllHosts(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllHosts")
	hostsList, err := ListHosts(ctx, apiClient, nil)
	if err != nil {
		return err
	}

	for hostID, hostUUID := range hostsList {
		zlog.Info().Msgf("Delete host %s %s", hostID, hostUUID)
		errDel := DeleteHost(ctx, apiClient, hostID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All hosts deleted")
	return nil
}

func DeleteAllInstances(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllInstances")
	instList, err := ListInstances(ctx, apiClient, nil)
	if err != nil {
		return err
	}

	for instID, hostID := range instList {
		zlog.Info().Msgf("Delete instance %s at %s", instID, hostID)
		errDel := DeleteInstance(ctx, apiClient, instID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All instances deleted")
	return nil
}

func HelperCleanupHosts(ctx context.Context, apiClient *api.ClientWithResponses) error {
	err := DeleteAllInstances(ctx, apiClient)
	if err != nil {
		return err
	}
	err = DeleteAllHosts(ctx, apiClient)
	if err != nil {
		return err
	}
	return nil
}

func HelperCleanupSchedules(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("Started helperCleanupSchedules")

	err := DeleteAllSingleScheds(ctx, apiClient)
	if err != nil {
		return err
	}

	err = DeleteAllRepeatedScheds(ctx, apiClient)
	if err != nil {
		return err
	}
	zlog.Info().Msg("Finished helperCleanupSchedules")
	return nil
}

func DeleteAllNonRootRegions(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllRegions")
	filter := fmt.Sprintf(`has(%s)`, "parent_region")
	wList, err := ListRegions(ctx, apiClient, &filter)
	if err != nil {
		return err
	}

	for itemID, itemField := range wList {
		zlog.Info().Msgf("Delete region %s Name %s", itemID, itemField)
		errDel := DeleteRegion(ctx, apiClient, itemID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All regions deleted")
	return nil
}

func HelperCleanupLocations(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("Started helperCleanupLocations")
	err := UnconfigureAllHosts(ctx, apiClient)
	if err != nil {
		return err
	}

	err = DeleteAllSites(ctx, apiClient)
	if err != nil {
		return err
	}

	err = DeleteAllNonRootRegions(ctx, apiClient)
	if err != nil {
		return err
	}

	err = DeleteAllRegions(ctx, apiClient)
	if err != nil {
		return err
	}
	zlog.Info().Msg("Finished helperCleanupLocations")
	return nil
}

func SetupRootRegions(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	maxRegions int,
	regionPrefixName string,
) {
	t.Helper()
	wg := &sync.WaitGroup{}
	counter := 1
	for r := 0; r < maxRegions; r++ {
		wg.Add(1)
		go func(r int) {
			defer wg.Done()

			req := Region1Request
			regName := fmt.Sprintf("%s-%d", regionPrefixName, r)
			req.Name = &regName
			req.ParentId = nil
			CreateRegion(ctx, t, apiClient, req)
			req.Name = &Region1Name
		}(r)

		// Wait for all requests in a batch to complete to keep going
		counter++
		if counter%requestsBatchSize == 0 {
			zlog.Info().Msgf("SetupRootRegions - Batch done %d of %d", counter, maxRegions)
			wg.Wait()
		}
	}
	wg.Wait()
}

func SetupSubRegions(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	maxSubRegions int,
	subRegionPrefixName string,
) {
	t.Helper()
	filter := fmt.Sprintf(`NOT has(%s)`, "parent_region")
	regions, err := ListRegions(ctx, apiClient, &filter)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	counter := 1

	for parentRegion := range regions {
		for sr := 0; sr < maxSubRegions; sr++ {
			wg.Add(1)
			go func(parentRegion string, sr int) {
				defer wg.Done()
				req := Region2Request
				subregName := fmt.Sprintf("%s-%d", subRegionPrefixName, sr)
				req.Name = &subregName
				req.ParentId = &parentRegion
				CreateRegion(ctx, t, apiClient, req)
				req.ParentId = nil
				req.Name = &Region2Name
			}(parentRegion, sr)

			// Wait for all requests in a batch to complete to keep going
			counter++
			if counter%requestsBatchSize == 0 {
				zlog.Info().Msgf("SetupSubRegions - Batch done %d of %d", counter, len(regions)*maxSubRegions)
				wg.Wait()
			}
		}
	}
	wg.Wait()
}

func SetupSites(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	maxSites int,
	sitePrefixName string,
) {
	t.Helper()
	filter := fmt.Sprintf(`has(%s)`, "parent_region")
	regions, err := ListRegions(ctx, apiClient, &filter)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	counter := 1

	for parentRegion := range regions {
		for si := 0; si < maxSites; si++ {
			wg.Add(1)
			go func(parentRegion string, si int) {
				defer wg.Done()
				req := Site2Request
				siteName := fmt.Sprintf("%s-%d", sitePrefixName, si)
				req.Name = &siteName
				req.RegionId = &parentRegion
				CreateSite(ctx, t, apiClient, req)
				req.Region = nil
			}(parentRegion, si)

			// Wait for all requests in a batch to complete to keep going
			counter++
			if counter%requestsBatchSize == 0 {
				zlog.Info().Msgf("SetupSites - Batch done %d of %d", counter, len(regions)*maxSites)
				wg.Wait()
			}
		}
	}
	wg.Wait()
}

func SetupRegionSiteLargeHierarchy(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	maxRegions,
	maxSubRegions,
	maxSites int,
	regionPrefixName,
	subRegionPrefixName,
	sitePrefixName string,
) {
	t.Helper()
	for r := 0; r < maxRegions; r++ {
		regName := fmt.Sprintf("%s-%d", regionPrefixName, r)
		Region1Request.Name = &regName
		Region1Request.ParentId = nil
		r1 := CreateRegion(ctx, t, apiClient, Region1Request)
		Region1Request.Name = &Region1Name

		for sr := 0; sr < maxSubRegions; sr++ {
			subregName := fmt.Sprintf("%s-%d-%d", subRegionPrefixName, r, sr)
			Region2Request.Name = &subregName
			Region2Request.ParentId = r1.JSON201.ResourceId
			r2 := CreateRegion(ctx, t, apiClient, Region2Request)
			Region2Request.ParentId = nil
			Region2Request.Name = &Region2Name

			for si := 0; si < maxSites; si++ {
				siteName := fmt.Sprintf("%s-%s-%d", subRegionPrefixName, sitePrefixName, si)
				Site2Request.Name = &siteName
				Site2Request.RegionId = r2.JSON201.RegionID
				CreateSite(ctx, t, apiClient, Site2Request)
				Site2Request.Region = nil
				Site2Request.Name = &Site2Name
			}
		}
	}
}

func SetupRegionSiteLargeHierarchyAsync(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	maxRegions,
	maxSubRegions,
	maxSites int,
	regionPrefixName,
	subRegionPrefixName,
	sitePrefixName string,
) {
	t.Helper()
	SetupRootRegions(
		ctx,
		t,
		apiClient,
		maxRegions,
		regionPrefixName,
	)

	SetupSubRegions(
		ctx,
		t,
		apiClient,
		maxSubRegions,
		subRegionPrefixName,
	)

	SetupSites(
		ctx,
		t,
		apiClient,
		maxSites,
		sitePrefixName,
	)
}

func ConfigureHostsLargeHierarchy(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
	hostsIDs []string,
) {
	t.Helper()
	allSites, err := ListSites(ctx, apiClient, nil)
	assert.NoError(t, err)

	assert.Equal(t, numSitesLargeHierarchy, len(allSites))
	assert.Equal(t, numHostsLargeHierarchy, len(hostsIDs))

	previous := 0
	hostsPerSite := 10
	for siteID := range allSites {
		siteHostsIDs := hostsIDs[previous : previous+hostsPerSite]
		zlog.Info().Msgf("ConfigureHostsLargeHierarchy - Current Batch %d of %d", previous, len(hostsIDs))
		err = ConfigureHostsbyID(ctx, apiClient, siteHostsIDs, siteID)
		zlog.Info().Msgf("ConfigureHostsLargeHierarchy - Batch done %d", hostsPerSite)
		assert.NoError(t, err)
		previous += hostsPerSite
	}
}

func SetHostsSingleSched(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses, hostIDs []string,
) {
	tb.Helper()
	wg := &sync.WaitGroup{}
	counter := 1
	for _, resID := range hostIDs {
		wg.Add(1)
		go func(resID string) {
			defer wg.Done()
			SschedName1 := singleSchedName
			now := time.Now().Unix()
			SschedStart1 := int(now + DelayStart5)

			SingleScheduleAlwaysRequest := api.SingleSchedule{
				Name:           &SschedName1,
				StartSeconds:   SschedStart1,
				ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
				TargetHostId:   &resID,
			}
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
			CreateSchedSingle(ctxInt, tb, apiClient, SingleScheduleAlwaysRequest)
		}(resID)

		// Wait for all requests in a batch to complete to keep going
		counter++
		if counter%requestsBatchSize == 0 {
			zlog.Info().Msgf("setHostsSingleSched - Batch done %d of %d", counter, len(hostIDs))
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
}

func SetSitesSingleSched(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses, siteIDs []string,
) {
	tb.Helper()
	for _, resID := range siteIDs {
		SschedName1 := singleSchedName
		now := time.Now().Unix()
		SschedStart1 := int(now + DelayStart5)
		SingleScheduleAlwaysRequest := api.SingleSchedule{
			Name:           &SschedName1,
			StartSeconds:   SschedStart1,
			ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
			TargetSiteId:   &resID,
		}
		CreateSchedSingle(ctx, tb, apiClient, SingleScheduleAlwaysRequest)
	}
}

func SetRegionSingleSched(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses, regionIDs []string,
) {
	tb.Helper()
	for _, resID := range regionIDs {
		SschedName1 := singleSchedName
		now := time.Now().Unix()
		SschedStart1 := int(now + DelayStart5)
		SingleScheduleAlwaysRequest := api.SingleSchedule{
			Name:           &SschedName1,
			StartSeconds:   SschedStart1,
			ScheduleStatus: api.SCHEDULESTATUSOSUPDATE,
			TargetRegionId: &resID,
		}
		CreateSchedSingle(ctx, tb, apiClient, SingleScheduleAlwaysRequest)
	}
}

func CheckHostsMaintenance(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses, hostIDs []string,
) {
	tb.Helper()
	wg := &sync.WaitGroup{}
	counter := 1
	chanErrs := make(chan error, len(hostIDs))
	for _, resID := range hostIDs {
		wg.Add(1)
		go func(resID string) {
			defer wg.Done()
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
			errHost := AssertHostInMaintenance(ctxInt, tb, apiClient, &resID, time.Now(), 1)
			if errHost != nil {
				chanErrs <- errHost
			}
		}(resID)

		// Wait for all requests in a batch to complete to keep going
		counter++
		if counter%requestsBatchSize == 0 {
			zlog.Info().Msgf("checkHostsMaintenance - Batch done of %d of total %d", counter, len(hostIDs))
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
	close(chanErrs)

	errors := []error{}
	for errH := range chanErrs {
		errors = append(errors, errH)
	}
	if len(errors) > 0 {
		err := fmt.Errorf("%v", errors)
		zlog.Error().Err(err).Msgf("failed checkHostsMaintenance for %d hosts", len(errors))
		assert.NoError(tb, err)
	}
}

func AssertHostInMaintenance(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	hostID *string,
	timestamp time.Time,
	expectedSchedules int,
) error {
	tb.Helper()

	timestampString := fmt.Sprint(timestamp.UTC().Unix())
	sReply, err := apiClient.GetSchedulesWithResponse(
		ctx,
		&api.GetSchedulesParams{
			HostID:    hostID,
			UnixEpoch: &timestampString,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to AssertHostInMaintenance - error in GetSchedulesWithResponse")
		return err
	}

	assert.Equal(tb, http.StatusOK, sReply.StatusCode())

	if http.StatusOK != sReply.StatusCode() {
		err = fmt.Errorf("http reply code not ok %d for host %s",
			sReply.StatusCode(), *hostID)
		zlog.Error().Err(err).Msg("failed to AssertHostInMaintenance")
		return err
	}

	numExistingSchedules := 0
	if sReply.JSON200.SingleSchedules != nil {
		numExistingSchedules += len(*sReply.JSON200.SingleSchedules)
	}
	if sReply.JSON200.RepeatedSchedules != nil {
		numExistingSchedules += len(*sReply.JSON200.RepeatedSchedules)
	}

	if numExistingSchedules != expectedSchedules {
		err = fmt.Errorf("wrong number of schedules numExistingSchedules %d expectedSchedules %d for host %s",
			numExistingSchedules, expectedSchedules, *hostID)
		zlog.Error().Err(err).Msg("failed to AssertHostInMaintenance")
		return err
	}

	return nil
}

func CheckSiteMaintenance(ctx context.Context, tb testing.TB, apiClient *api.ClientWithResponses, siteID string) {
	tb.Helper()
	ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()
	AssertInMaintenance(ctxInt, tb, apiClient, nil, &siteID, nil, time.Now(), 1, true)
}

func CheckRegionMaintenance(ctx context.Context, tb testing.TB, apiClient *api.ClientWithResponses, regionID string) {
	tb.Helper()
	ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()
	AssertInMaintenance(ctxInt, tb, apiClient, nil, nil, &regionID, time.Now(), 1, true)
}

func ConfigureHostsbyID(
	ctx context.Context,
	apiClient *api.ClientWithResponses,
	hostsIDs []string, siteID string,
) error {
	chanErrs := make(chan error, len(hostsIDs))
	wg := &sync.WaitGroup{}
	counter := 1

	for _, resID := range hostsIDs {
		wg.Add(1)
		go func(resID string) {
			defer wg.Done()
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()

			err := configureHostbyID(ctxInt, apiClient, resID, resID, siteID)
			if err != nil {
				chanErrs <- err
			}
		}(resID)

		counter++
		if counter%requestsBatchSize == 0 {
			zlog.Info().Msgf("configureHostsbyID - Batch done of %d of total %d", counter, len(hostsIDs))
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
	close(chanErrs)

	err := CheckErrors(chanErrs)
	if err != nil {
		zlog.Error().Err(err).Msg("failed configureHostsbyID")
		return err
	}
	return nil
}

func configureHostbyID(
	ctx context.Context,
	apiClient *api.ClientWithResponses,
	hostID, hostName, siteID string,
) error {
	if hostName == "" {
		hostName = hostUUID
		zlog.Info().Msgf("Configuring hostID %s name %s to siteID %s", hostUUID, hostName, siteID)
	}
	hostRequestPatch := api.Host{Name: hostName, SiteId: &siteID}
	res, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		hostID,
		hostRequestPatch,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusOK, res.StatusCode())
	if err != nil {
		return err
	}

	return nil
}

func SetAllRootRegionsInSingleSched(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
) {
	t.Helper()

	filter := fmt.Sprintf(`NOT has(%s)`, "parent_region")
	regions, err := ListRegions(ctx, apiClient, &filter)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	counter := 1

	for region := range regions {
		wg.Add(1)
		go func(resID string) {
			defer wg.Done()
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
			SetRegionSingleSched(
				ctxInt,
				t,
				apiClient, []string{resID})
		}(region)

		counter++
		if counter%requestsBatchSize == 0 {
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
}

func CheckAllRegionsInMaintenance(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
) {
	t.Helper()
	regions, err := ListRegions(ctx, apiClient, nil)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	counter := 1

	for region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
			AssertInMaintenance(ctxInt, t, apiClient, nil, nil, &region, time.Now(), 1, true)
		}(region)
		counter++
		if counter%requestsBatchSize == 0 {
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
}

func CheckAllSitesInMaintenance(
	ctx context.Context,
	t *testing.T,
	apiClient *api.ClientWithResponses,
) {
	t.Helper()
	sites, err := ListSites(ctx, apiClient, nil)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	counter := 1

	for site := range sites {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			ctxInt, cancel := context.WithTimeout(ctx, ctxTimeout)
			defer cancel()
			AssertInMaintenance(ctxInt, t, apiClient, nil, &site, nil, time.Now(), 1, true)
		}(site)
		counter++
		if counter%requestsBatchSize == 0 {
			wg.Wait()
			time.Sleep(requestsInterval)
		}
	}
	wg.Wait()
}

func UnallocateHostFromSite(ctx context.Context, apiClient *api.ClientWithResponses, hostID string) error {
	hostRequestPatch := api.Host{
		SiteId: &emptyString,
	}
	res, err := apiClient.PatchComputeHostsHostIDWithResponse(
		ctx,
		hostID,
		hostRequestPatch,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusOK, res.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func UnconfigureAllHosts(ctx context.Context, apiClient *api.ClientWithResponses) error {
	filter := "has(site)"
	hostsList, err := ListHosts(ctx, apiClient, &filter)
	if err != nil {
		return err
	}
	zlog.Info().Msgf("UnconfigureAllHosts %d", len(hostsList))

	wg := &sync.WaitGroup{}
	chanErrs := make(chan error, len(hostsList))
	counter := 1
	for hostID, hostUUID := range hostsList {
		wg.Add(1)
		go func(hostID, hostUUID string) {
			defer wg.Done()
			zlog.Info().Msgf("Unconfigure host %s %s", hostID, hostUUID)
			errUnconfig := UnallocateHostFromSite(ctx, apiClient, hostID)
			if errUnconfig != nil {
				chanErrs <- errUnconfig
			}
		}(hostID, hostUUID)
		counter++
		if counter%batchRequestsSize == 0 {
			wg.Wait()
			time.Sleep(rateLimitInterval)
		}
	}

	wg.Wait()
	close(chanErrs)

	err = CheckErrors(chanErrs)
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to set all hosts unconfigured")
		return err
	}
	zlog.Info().Msgf("All hosts unconfigured")
	return nil
}

// ListSites returns a map of all sites indexed by siteID associated with siteName.
func ListSites(ctx context.Context, apiClient *api.ClientWithResponses, filter *string) (map[string]string, error) {
	zlog.Info().Msg("ListSites")
	sitesList := make(map[string]string)

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetSitesWithResponse(
			ctx,
			&api.GetSitesParams{
				Filter:   filter,
				Offset:   &offset,
				PageSize: &pageSize,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list sites")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		for _, site := range *resList.JSON200.Sites {
			sitesList[*site.ResourceId] = *site.Name
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All sites listed %v", sitesList)

			break
		}
		offset += pageSize
	}
	return sitesList, nil
}

// ListRegions returns a map of all regions indexed by regionID associated with regionName.
func ListRegions(ctx context.Context, apiClient *api.ClientWithResponses, filter *string) (map[string]string, error) {
	zlog.Info().Msg("ListRegions")
	regionsList := make(map[string]string)

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetRegionsWithResponse(
			ctx,
			&api.GetRegionsParams{
				Filter:   filter,
				Offset:   &offset,
				PageSize: &pageSize,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list regions")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		for _, region := range *resList.JSON200.Regions {
			regionsList[*region.ResourceId] = *region.Name
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All regions listed %v", regionsList)

			break
		}
		offset += pageSize
	}
	return regionsList, nil
}

func DeleteAllSites(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllSites")
	wList, err := ListSites(ctx, apiClient, nil)
	if err != nil {
		return err
	}

	for itemID, itemField := range wList {
		zlog.Info().Msgf("Delete site %s name %s", itemID, itemField)
		errDel := DeleteSite(ctx, apiClient, itemID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All sites deleted")
	return nil
}

func DeleteAllRegions(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllRegions")
	wList, err := ListRegions(ctx, apiClient, nil)
	if err != nil {
		return err
	}

	for itemID, itemField := range wList {
		zlog.Info().Msgf("Delete region %s Name %s", itemID, itemField)
		errDel := DeleteRegion(ctx, apiClient, itemID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All regions deleted")
	return nil
}

func DeleteRegion(
	ctx context.Context,
	apiClient *api.ClientWithResponses,
	regionID string,
) error {
	resDelRegion, err := apiClient.DeleteRegionsRegionIDWithResponse(
		ctx,
		regionID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelRegion.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func DeleteSite(
	ctx context.Context,
	apiClient *api.ClientWithResponses,
	siteID string,
) error {
	resDelSite, err := apiClient.DeleteSitesSiteIDWithResponse(ctx,
		siteID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelSite.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

// ListSinleScheds returns a map of all single scheds indexed by schedID associated with struct{}.
func ListSingleScheds(ctx context.Context, apiClient *api.ClientWithResponses) (map[string]struct{}, error) {
	zlog.Info().Msg("ListSingleScheds")
	schedList := make(map[string]struct{})

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetSchedulesSingleWithResponse(
			ctx,
			&api.GetSchedulesSingleParams{
				Offset:   &offset,
				PageSize: &pageSize,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list instances")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		if resList.JSON200.SingleSchedules != nil {
			for _, res := range *resList.JSON200.SingleSchedules {
				schedList[*res.ResourceId] = struct{}{}
			}
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All single scheds listed %v", schedList)

			break
		}

		offset += pageSize
	}
	return schedList, nil
}

// ListRepeatedScheds returns a map of all repeated scheds indexed by schedID associated with struct{}.
func ListRepeatedScheds(ctx context.Context, apiClient *api.ClientWithResponses) (map[string]struct{}, error) {
	zlog.Info().Msg("ListRepeatedScheds")
	schedList := make(map[string]struct{})

	offset := 0
	pageSize := 100
	for {
		resList, err := apiClient.GetSchedulesRepeatedWithResponse(
			ctx,
			&api.GetSchedulesRepeatedParams{
				Offset:   &offset,
				PageSize: &pageSize,
			},
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to list instances")
			return nil, err
		}
		err = CheckResponse(http.StatusOK, resList.StatusCode())
		if err != nil {
			return nil, err
		}

		if resList.JSON200.RepeatedSchedules != nil {
			for _, res := range *resList.JSON200.RepeatedSchedules {
				schedList[*res.ResourceId] = struct{}{}
			}
		}

		if !*resList.JSON200.HasNext {
			zlog.Info().Msgf("All repeated scheds listed %v", schedList)

			break
		}

		offset += pageSize
	}
	return schedList, nil
}

func DeleteSingleSched(ctx context.Context, apiClient *api.ClientWithResponses, itemID string) error {
	resDelRegion, err := apiClient.DeleteSchedulesSingleSingleScheduleIDWithResponse(
		ctx,
		itemID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelRegion.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllSingleScheds(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllSingleScheds")
	wList, err := ListSingleScheds(ctx, apiClient)
	if err != nil {
		return err
	}

	for itemID := range wList {
		zlog.Info().Msgf("Delete single sched %s ", itemID)
		errDel := DeleteSingleSched(ctx, apiClient, itemID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All single scheds deleted")
	return nil
}

func DeleteRepeatedSched(ctx context.Context, apiClient *api.ClientWithResponses, itemID string) error {
	resDelRegion, err := apiClient.DeleteSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		itemID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	err = CheckResponse(http.StatusNoContent, resDelRegion.StatusCode())
	if err != nil {
		return err
	}
	return nil
}

func DeleteAllRepeatedScheds(ctx context.Context, apiClient *api.ClientWithResponses) error {
	zlog.Info().Msg("DeleteAllRepeatedScheds")
	wList, err := ListRepeatedScheds(ctx, apiClient)
	if err != nil {
		return err
	}

	for itemID := range wList {
		zlog.Info().Msgf("Delete single sched %s ", itemID)
		errDel := DeleteRepeatedSched(ctx, apiClient, itemID)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All repeated scheds deleted")
	return nil
}

func CheckHostStatus(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	hostUUID string,
) {
	tb.Helper()
	filterUUID := fmt.Sprintf(`%s = %q`, "uuid", hostUUID)
	hostIDs, err := ListHosts(ctx, apiClient, &filterUUID)
	require.NoError(tb, err)

	for hostID := range hostIDs {
		hostResp, errHost := apiClient.GetComputeHostsHostIDWithResponse(ctx, hostID,
			utils.AddJWTtoTheHeader, utils.AddProjectIDtoTheHeader)
		require.NoError(tb, errHost)
		require.NotNil(tb, hostResp)

		host := hostResp.JSON200
		if host != nil {
			assert.Equal(tb, *host.HostStatus,
				host_status.HostStatusRunning.Status)
			assert.Equal(tb, string(*host.HostStatusIndicator),
				host_status.HostStatusRunning.StatusIndicator.String())
			assert.Equal(tb, *host.OnboardingStatus,
				onb_status.OnboardingStatusDone.Status)
			assert.Equal(tb, string(*host.OnboardingStatusIndicator),
				onb_status.OnboardingStatusDone.StatusIndicator.String())
			if host.Instance != nil {
				assert.Equal(tb, *host.Instance.InstanceStatus,
					host_status.InstanceStatusRunning.Status)
				assert.Equal(tb, string(*host.Instance.InstanceStatusIndicator),
					host_status.InstanceStatusRunning.StatusIndicator.String())
				assert.Equal(tb, *host.Instance.ProvisioningStatus,
					onb_status.ProvisioningStatusDone.Status)
				assert.Equal(tb, string(*host.Instance.ProvisioningStatusIndicator),
					onb_status.ProvisioningStatusDone.StatusIndicator.String())
				assert.Equal(tb, *host.Instance.UpdateStatus,
					maint_status.UpdateStatusUpToDate.Status)
				assert.Equal(tb, string(*host.Instance.UpdateStatusIndicator),
					maint_status.UpdateStatusUpToDate.StatusIndicator.String())
			}
		}
	}
}

func UpdateHostOS(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	hostID string,
	updateSources []string,
	installedPkgs string,
	kernelCmd string,
) {
	tb.Helper()

	hostResp, errHost := apiClient.GetComputeHostsHostIDWithResponse(
		ctx,
		hostID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, errHost)
	require.Equal(tb, http.StatusOK, hostResp.StatusCode())
	require.NotNil(tb, hostResp)
	require.NotNil(tb, hostResp.JSON200.Instance)

	osID := *hostResp.JSON200.Instance.Os.ResourceId
	osSHA := hostResp.JSON200.Instance.Os.Sha256
	osBody := api.OperatingSystemResource{
		UpdateSources:     updateSources,
		InstalledPackages: &installedPkgs,
		Sha256:            osSHA,
		KernelCommand:     &kernelCmd,
	}
	osResp, errOs := apiClient.PatchOSResourcesOSResourceIDWithResponse(
		ctx,
		osID,
		osBody,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, errOs)
	require.Equal(tb, http.StatusOK, osResp.StatusCode())
}

func CreateRegion(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	regionRequest api.Region,
) *api.PostRegionsResponse {
	tb.Helper()

	region, err := apiClient.PostRegionsWithResponse(ctx, regionRequest, utils.AddJWTtoTheHeader, utils.AddProjectIDtoTheHeader)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusCreated, region.StatusCode())

	tb.Cleanup(func() {
		err := DeleteRegion(context.Background(), apiClient, *region.JSON201.RegionID)
		assert.NoError(tb, err)
	})
	return region
}

func CreateSite(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	siteRequest api.Site,
) *api.PostSitesResponse {
	tb.Helper()

	site, err := apiClient.PostSitesWithResponse(ctx, siteRequest, utils.AddJWTtoTheHeader, utils.AddProjectIDtoTheHeader)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusCreated, site.StatusCode())

	tb.Cleanup(func() {
		err := DeleteSite(context.Background(), apiClient, *site.JSON201.ResourceId)
		assert.NoError(tb, err)
	})
	return site
}

func CreateSchedSingle(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	reqSched api.SingleSchedule,
) *api.PostSchedulesSingleResponse {
	tb.Helper()

	sched, err := apiClient.PostSchedulesSingleWithResponse(
		ctx,
		reqSched,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusCreated, sched.StatusCode())

	tb.Cleanup(func() { DeleteSchedSingle(context.Background(), tb, apiClient, *sched.JSON201.SingleScheduleID) })
	return sched
}

func DeleteSchedSingle(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	schedID string,
) {
	tb.Helper()

	schedDel, err := apiClient.DeleteSchedulesSingleSingleScheduleIDWithResponse(
		ctx,
		schedID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusNoContent, schedDel.StatusCode())
}

func CreateSchedRepeated(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	reqSched api.RepeatedSchedule,
) *api.PostSchedulesRepeatedResponse {
	tb.Helper()

	sched, err := apiClient.PostSchedulesRepeatedWithResponse(
		ctx,
		reqSched,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusCreated, sched.StatusCode())

	tb.Cleanup(func() { DeleteSchedRepeated(context.Background(), tb, apiClient, *sched.JSON201.RepeatedScheduleID) })
	return sched
}

func DeleteSchedRepeated(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	schedID string,
) {
	tb.Helper()

	schedDel, err := apiClient.DeleteSchedulesRepeatedRepeatedScheduleIDWithResponse(
		ctx,
		schedID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, err)
	assert.Equal(tb, http.StatusNoContent, schedDel.StatusCode())
}

func AssertInMaintenance(
	ctx context.Context,
	tb testing.TB,
	apiClient *api.ClientWithResponses,
	hostID *string,
	siteID *string,
	regionID *string,
	timestamp time.Time,
	expectedSchedules int,
	found bool,
) {
	tb.Helper()

	timestampString := fmt.Sprint(timestamp.UTC().Unix())
	sReply, err := apiClient.GetSchedulesWithResponse(
		ctx,
		&api.GetSchedulesParams{
			HostID:    hostID,
			SiteID:    siteID,
			RegionID:  regionID,
			UnixEpoch: &timestampString,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	require.NoError(tb, err)
	if found {
		assert.Equal(tb, http.StatusOK, sReply.StatusCode())
		length := 0
		if sReply.JSON200.SingleSchedules != nil {
			length += len(*sReply.JSON200.SingleSchedules)
		}
		if sReply.JSON200.RepeatedSchedules != nil {
			length += len(*sReply.JSON200.RepeatedSchedules)
		}
		assert.Equal(tb, expectedSchedules, length, "Wrong number of schedules")
	} else {
		assert.Equal(tb, http.StatusOK, sReply.StatusCode())
	}
}

func APICheckHosts(ctx context.Context,
	apiClient *http.Client,
	url string,
	filter *string,
	amount int,
) error {
	listItems, err := ListHostsAPI(ctx, apiClient, url, filter)
	if err != nil {
		return err
	}

	if amount != len(listItems) {
		return fmt.Errorf("expected %d, got %d", amount, len(listItems))
	}

	return nil
}

func APICheckInstances(ctx context.Context,
	apiClient *http.Client,
	url string,
	filter *string,
	amount int,
) error {
	listItems, err := ListInstancesAPI(ctx, apiClient, url, filter)
	if err != nil {
		return err
	}

	if amount != len(listItems) {
		return fmt.Errorf("expected %d, got %d", amount, len(listItems))
	}

	return nil
}

func InfraAPICheckHosts(ctx context.Context,
	apiClient *api.ClientWithResponses,
	filter *string,
	amount int,
) error {
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)

	if http.StatusOK != resList.StatusCode() {
		return err
	}
	if amount != *resList.JSON200.TotalElements {
		return err
	}
	return nil
}

func InfraAPICheckInstances(ctx context.Context,
	apiClient *api.ClientWithResponses,
	filter *string,
	amount int,
) error {
	resList, err := apiClient.GetInstancesWithResponse(
		ctx,
		&api.GetInstancesParams{
			Filter: filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)

	if http.StatusOK != resList.StatusCode() {
		return err
	}
	if amount != *resList.JSON200.TotalElements {
		return err
	}
	return nil
}
