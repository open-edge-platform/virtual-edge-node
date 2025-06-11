// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	edgeinfraapi "github.com/open-edge-platform/infra-core/apiv2/v2/pkg/api/v2"
	host_status "github.com/open-edge-platform/infra-managers/host/pkg/status"
	maint_status "github.com/open-edge-platform/infra-managers/maintenance/pkg/status"
	onb_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
)

const (
	// wait time for HTTP request.
	waitTime = 60 * time.Second
)

var ErrListedAllSentinel = fmt.Errorf("all elements listed")

var (
	hostsURL     = "https://api.%s/v1/projects/%s/compute/hosts"
	instancesURL = "https://api.%s/v1/projects/%s/compute/instances"
)

func getToken() (string, error) {
	jwtTokenStr, ok := os.LookupEnv(ENVJWTToken)
	if !ok {
		return "", fmt.Errorf("can't find a \"JWT_TOKEN\" variable, please set it in your environment")
	}

	return jwtTokenStr, nil
}

func httpGet(ctx context.Context, client *http.Client, url string,
	responseHooker func(*http.Response) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	token, err := getToken()
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP GET to %s failed, status: %s", url, resp.Status)
		return err
	}

	if responseHooker != nil {
		if err := responseHooker(resp); err != nil {
			return err
		}
	}

	return nil
}

func httpDelete(ctx context.Context, client *http.Client,
	url string,
	responseHooker func(*http.Response) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		url, http.NoBody)
	if err != nil {
		return err
	}
	token, err := getToken()
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP Delete to %s failed, status: %s",
			url, resp.Status)
		return err
	}

	if responseHooker != nil {
		if err := responseHooker(resp); err != nil {
			return err
		}
	}

	return nil
}

func ListHostsAPI(ctx context.Context, apiClient *http.Client,
	url string,
	filter *string,
) (map[string]string, error) {
	zlog.Info().Msg("ListHosts")
	hostsList := make(map[string]string)

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListHostsResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Hosts) == 0 {
			return ErrListedAllSentinel
		}
		for _, host := range ps.Hosts {
			hostsList[*host.ResourceId] = *host.Uuid
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", hostsList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)
		if filter != nil {
			fmtURL = fmt.Sprintf("%s&filter=%s", fmtURL, *filter)
		}

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return hostsList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return hostsList, nil
}

func ListInstancesAPI(ctx context.Context, apiClient *http.Client,
	url string,
	filter *string,
) (map[string]string, error) {
	zlog.Info().Msg("ListInstances")
	resList := make(map[string]string)

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListInstancesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Instances) == 0 {
			return ErrListedAllSentinel
		}
		for _, res := range ps.Instances {
			resList[*res.ResourceId] = *res.Host.ResourceId
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", resList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)
		if filter != nil {
			fmtURL = fmt.Sprintf("%s&filter=%s", fmtURL, *filter)
		}

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return resList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return resList, nil
}

func DeleteResourceAPI(ctx context.Context,
	client *http.Client,
	url string,
) error {
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()
	if err := httpDelete(rCtx, client, url, nil); err != nil {
		return err
	}

	return nil
}

func DeleteAllHostsAPI(ctx context.Context, client *http.Client, cfg *flags_test.TestConfig, filter *string) error {
	zlog.Info().Msg("DeleteAllHosts")

	fmtHostsURL := fmt.Sprintf(hostsURL, cfg.OrchFQDN, cfg.Project)
	hostsList, err := ListHostsAPI(ctx, client, fmtHostsURL, filter)
	if err != nil {
		return err
	}

	for hostID, hostUUID := range hostsList {
		zlog.Info().Msgf("Delete host %s %s", hostID, hostUUID)
		delHostURL := fmt.Sprintf("%s/%s", fmtHostsURL, hostID)
		errDel := DeleteResourceAPI(ctx, client, delHostURL)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All hosts deleted")
	return nil
}

func DeleteAllInstancesAPI(ctx context.Context, client *http.Client, cfg *flags_test.TestConfig, filter *string) error {
	zlog.Info().Msg("DeleteAllInstances")
	fmtInstancesURL := fmt.Sprintf(instancesURL, cfg.OrchFQDN, cfg.Project)
	instList, err := ListInstancesAPI(ctx, client, fmtInstancesURL, filter)
	if err != nil {
		return err
	}

	for instID, hostID := range instList {
		zlog.Info().Msgf("Delete instance %s at %s", instID, hostID)
		delInstanceURL := fmt.Sprintf("%s/%s", fmtInstancesURL, instID)
		errDel := DeleteResourceAPI(ctx, client, delInstanceURL)
		if errDel != nil {
			return errDel
		}
	}
	zlog.Info().Msgf("All instances deleted")
	return nil
}

// HelperCleanupHostsAPI cleans up all hosts and instances in the Infrastructure Manager.
func HelperCleanupHostsAPI(_ context.Context, client *http.Client, cfg *flags_test.TestConfig) error {
	err := DeleteAllInstancesAPI(context.Background(), client, cfg, nil)
	if err != nil {
		return err
	}
	err = DeleteAllHostsAPI(context.Background(), client, cfg, nil)
	if err != nil {
		return err
	}
	return nil
}

// ListSingleSchedulesAPI lists all single schedules in the Infrastructure Manager.
func ListSingleSchedulesAPI(ctx context.Context, apiClient *http.Client, url string) (map[string]struct{}, error) {
	zlog.Info().Msg("ListSingleSchedules")
	schedulesList := make(map[string]struct{})

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListSingleSchedulesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.SingleSchedules) == 0 {
			return ErrListedAllSentinel
		}
		for _, schedule := range ps.SingleSchedules {
			schedulesList[*schedule.ResourceId] = struct{}{}
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", schedulesList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return schedulesList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return schedulesList, nil
}

// ListRepeatedSchedulesAPI lists all repeated schedules in the Infrastructure Manager.
func ListRepeatedSchedulesAPI(ctx context.Context, apiClient *http.Client, url string) (map[string]struct{}, error) {
	zlog.Info().Msg("ListRepeatedSchedules")
	schedulesList := make(map[string]struct{})

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListRepeatedSchedulesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.RepeatedSchedules) == 0 {
			return ErrListedAllSentinel
		}
		for _, schedule := range ps.RepeatedSchedules {
			schedulesList[*schedule.ResourceId] = struct{}{}
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", schedulesList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return schedulesList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return schedulesList, nil
}

// ListRegionsAPI lists all regions in the Infrastructure Manager.
func ListRegionsAPI(ctx context.Context, apiClient *http.Client, url string) (map[string]struct{}, error) {
	zlog.Info().Msg("ListRegions")
	regionsList := make(map[string]struct{})

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListRegionsResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Regions) == 0 {
			return ErrListedAllSentinel
		}
		for _, region := range ps.Regions {
			regionsList[*region.ResourceId] = struct{}{}
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", regionsList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return regionsList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return regionsList, nil
}

// ListSitesAPI lists all sites in the Infrastructure Manager.
func ListSitesAPI(ctx context.Context, apiClient *http.Client, url string) (map[string]struct{}, error) {
	zlog.Info().Msg("ListSites")
	sitesList := make(map[string]struct{})

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListSitesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Sites) == 0 {
			return ErrListedAllSentinel
		}
		for _, site := range ps.Sites {
			sitesList[*site.ResourceId] = struct{}{}
		}

		if !ps.HasNext {
			zlog.Info().Msgf("All listed %v", sitesList)
			return ErrListedAllSentinel
		}
		return nil
	}

	offset := 0
	pageSize := 100
	for {
		fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", url, offset, pageSize)

		rCtx, cancel := context.WithTimeout(ctx, waitTime)
		err := httpGet(rCtx, apiClient, fmtURL, responseHooker)
		if err != nil && !errors.Is(err, ErrListedAllSentinel) {
			cancel()
			return sitesList, err
		}
		cancel()

		if errors.Is(err, ErrListedAllSentinel) {
			zlog.Info().Msg("All listed")
			break
		}

		offset += pageSize
	}
	return sitesList, nil
}

// HelperCleanupLocationsAPI cleans up all locations in the Infrastructure Manager.
// Lists all regions and sites and deletes them.
func HelperCleanupLocationsAPI(_ context.Context, client *http.Client, cfg *flags_test.TestConfig) error {
	// List and delete all regions
	regionsURL := fmt.Sprintf("https://api.%s/v1/projects/%s/regions", cfg.OrchFQDN, cfg.Project)
	regionsList, err := ListRegionsAPI(context.Background(), client, regionsURL)
	if err != nil {
		return err
	}
	for regionID := range regionsList {
		delRegionURL := fmt.Sprintf("%s/%s", regionsURL, regionID)
		if errDel := DeleteResourceAPI(context.Background(), client, delRegionURL); errDel != nil {
			return errDel
		}
	}

	// List and delete all sites
	sitesURL := fmt.Sprintf("https://api.%s/v1/projects/regions/region-12345678/%s/sites", cfg.OrchFQDN, cfg.Project)
	sitesList, err := ListSitesAPI(context.Background(), client, sitesURL)
	if err != nil {
		return err
	}
	for siteID := range sitesList {
		delSiteURL := fmt.Sprintf("%s/%s", sitesURL, siteID)
		if errDel := DeleteResourceAPI(context.Background(), client, delSiteURL); errDel != nil {
			return errDel
		}
	}

	return nil
}

// HelperCleanupSchedulesAPI cleans up all single and repeated schedules in the Infrastructure Manager.
// Lists all single and repeated schedules and deletes them.
func HelperCleanupSchedulesAPI(_ context.Context, client *http.Client, cfg *flags_test.TestConfig) error {
	singleSchedulesURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules/single", cfg.OrchFQDN, cfg.Project)
	singleSchedulesList, err := ListSingleSchedulesAPI(context.Background(), client, singleSchedulesURL)
	if err != nil {
		return err
	}
	for scheduleID := range singleSchedulesList {
		delScheduleURL := fmt.Sprintf("%s/%s", singleSchedulesURL, scheduleID)
		if errDel := DeleteResourceAPI(context.Background(), client, delScheduleURL); errDel != nil {
			return errDel
		}
	}

	repeatedSchedulesURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules/repeated", cfg.OrchFQDN, cfg.Project)
	repeatedSchedulesList, err := ListRepeatedSchedulesAPI(context.Background(), client, repeatedSchedulesURL)
	if err != nil {
		return err
	}
	for scheduleID := range repeatedSchedulesList {
		delScheduleURL := fmt.Sprintf("%s/%s", repeatedSchedulesURL, scheduleID)
		if errDel := DeleteResourceAPI(context.Background(), client, delScheduleURL); errDel != nil {
			return errDel
		}
	}

	return nil
}

func ListHostsTotalAPI(ctx context.Context, apiClient *http.Client,
	cfg *flags_test.TestConfig,
	filter *string,
) (int, error) {
	zlog.Info().Msg("ListHostsTotalAPI")
	fmtHostsURL := fmt.Sprintf(hostsURL, cfg.OrchFQDN, cfg.Project)
	totalElements := 0
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListHostsResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		totalElements = int(ps.TotalElements)
		return nil
	}

	offset := 0
	pageSize := 100
	fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", fmtHostsURL, offset, pageSize)
	if filter != nil {
		fmtURL = fmt.Sprintf("%s&filter=%s", fmtURL, *filter)
	}

	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	if err := httpGet(rCtx, apiClient, fmtURL, responseHooker); err != nil {
		cancel()
		return totalElements, err
	}
	cancel()

	return totalElements, nil
}

func ListInstancesTotalAPI(ctx context.Context, apiClient *http.Client,
	cfg *flags_test.TestConfig,
	filter *string,
) (int, error) {
	zlog.Info().Msg("ListInstancesTotalAPI")
	fmtInstancesURL := fmt.Sprintf(instancesURL, cfg.OrchFQDN, cfg.Project)
	totalElements := 0
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListInstancesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		totalElements = int(ps.TotalElements)
		return nil
	}

	offset := 0
	pageSize := 100
	fmtURL := fmt.Sprintf("%s?offset=%d&pageSize=%d", fmtInstancesURL, offset, pageSize)
	if filter != nil {
		fmtURL = fmt.Sprintf("%s&filter=%s", fmtURL, *filter)
	}

	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	if err := httpGet(rCtx, apiClient, fmtURL, responseHooker); err != nil {
		cancel()
		return totalElements, err
	}
	cancel()

	return totalElements, nil
}

func getHostAPI(ctx context.Context, apiClient *http.Client,
	cfg *flags_test.TestConfig,
	hostID string,
) (*edgeinfraapi.HostResource, error) {
	zlog.Info().Msg("getHostAPI")
	fmtHostsURL := fmt.Sprintf(hostsURL, cfg.OrchFQDN, cfg.Project)
	hostReply := &edgeinfraapi.HostResource{}
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.GetHostResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		hostReply = &ps.Host
		return nil
	}

	fmtURL := fmt.Sprintf("%s/%s", fmtHostsURL, hostID)
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	if err := httpGet(rCtx, apiClient, fmtURL, responseHooker); err != nil {
		cancel()
		return hostReply, err
	}
	cancel()

	return hostReply, nil
}

func CheckHostStatusAPI(
	ctx context.Context,
	tb testing.TB,
	apiClient *http.Client,
	cfg *flags_test.TestConfig,
	hostUUID string,
) {
	tb.Helper()
	fmtHostsURL := fmt.Sprintf(hostsURL, cfg.OrchFQDN, cfg.Project)
	filterUUID := fmt.Sprintf(`%s = %q`, "uuid", hostUUID)
	hostIDs, err := ListHostsAPI(ctx, apiClient, fmtHostsURL, &filterUUID)
	require.NoError(tb, err)

	for hostID := range hostIDs {
		host, errHost := getHostAPI(ctx, apiClient, cfg, hostID)
		require.NoError(tb, errHost)
		require.NotNil(tb, host)

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

// ListHosts returns a map of hostID to hostUUID using the HTTP API.
func ListHosts(ctx context.Context, apiClient *http.Client, filter *string) (map[string]string, error) {
	fmtHostsURL := fmt.Sprintf(hostsURL, flags_test.GetConfig().OrchFQDN, flags_test.GetConfig().Project)
	return ListHostsAPI(ctx, apiClient, fmtHostsURL, filter)
}

// ListInstances returns a map of instanceID to hostID using the HTTP API.
func ListInstances(ctx context.Context, apiClient *http.Client, filter *string) (map[string]string, error) {
	fmtInstancesURL := fmt.Sprintf(instancesURL, flags_test.GetConfig().OrchFQDN, flags_test.GetConfig().Project)
	return ListInstancesAPI(ctx, apiClient, fmtInstancesURL, filter)
}

// UpdateHostOS updates the OS of a host using PATCH request.
func UpdateHostOS(ctx context.Context, tb testing.TB,
	apiClient *http.Client, hostID string,
	updateSources []string,
	installedPkgs, kernelCmd string,
) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	fmtHostsURL := fmt.Sprintf(hostsURL, cfg.OrchFQDN, cfg.Project)
	getURL := fmt.Sprintf("%s/%s", fmtHostsURL, hostID)
	var osID string
	var osSHA *string
	var instance *edgeinfraapi.InstanceResource

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.GetHostResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if ps.Host.Instance == nil || ps.Host.Instance.Os == nil || ps.Host.Instance.Os.ResourceId == nil {
			return fmt.Errorf("host instance or os is nil")
		}
		osID = *ps.Host.Instance.Os.ResourceId
		osSHA = &ps.Host.Instance.Os.Sha256
		instance = ps.Host.Instance
		return nil
	}
	err := httpGet(ctx, apiClient, getURL, responseHooker)
	if err != nil {
		tb.Fatalf("failed to get host: %v", err)
	}
	if instance == nil {
		tb.Fatalf("host instance is nil")
	}
	var sha string
	if osSHA != nil {
		sha = *osSHA
	}
	osBody := edgeinfraapi.OperatingSystemResource{
		UpdateSources:     &updateSources,
		InstalledPackages: &installedPkgs,
		Sha256:            sha,
		KernelCommand:     &kernelCmd,
	}
	osBodyBytes, err := json.Marshal(osBody)
	if err != nil {
		tb.Fatalf("failed to marshall patch request: %v", err)
	}
	osURL := fmt.Sprintf("https://api.%s/v1/projects/%s/compute/os/%s", cfg.OrchFQDN, cfg.Project, osID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, osURL, io.NopCloser(strings.NewReader(string(osBodyBytes))))
	if err != nil {
		tb.Fatalf("failed to create patch request: %v", err)
	}
	token, err := getToken()
	if err != nil {
		tb.Fatalf("failed to get token: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := apiClient.Do(req)
	if err != nil {
		tb.Fatalf("failed to patch os: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tb.Fatalf("patch os failed: %s", resp.Status)
	}
}

// SetHostsSingleSched sets a single schedule for each hostID.
func SetHostsSingleSched(ctx context.Context, tb testing.TB, apiClient *http.Client, hostIDs []string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules/single", cfg.OrchFQDN, cfg.Project)
	for _, hostID := range hostIDs {
		now := time.Now().Unix()
		schedStart := int(now + DelayStart5) // DelayStart5
		body := map[string]interface{}{
			"name":           schedName,
			"startSeconds":   schedStart,
			"scheduleStatus": "OS_UPDATE",
			"targetHostId":   hostID,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			tb.Fatalf("failed to marshall schedule request: %v", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, schedURL, io.NopCloser(strings.NewReader(string(bodyBytes))))
		if err != nil {
			tb.Fatalf("failed to create schedule request: %v", err)
		}
		token, err := getToken()
		if err != nil {
			tb.Fatalf("failed to get token: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := apiClient.Do(req)
		if err != nil {
			tb.Fatalf("failed to post schedule: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			tb.Fatalf("post schedule failed: %s", resp.Status)
		}
	}
}

// SetSitesSingleSched sets a single schedule for each siteID.
func SetSitesSingleSched(ctx context.Context, tb testing.TB, apiClient *http.Client, siteIDs []string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules/single", cfg.OrchFQDN, cfg.Project)
	for _, siteID := range siteIDs {
		schedName := "schedSingle"
		now := time.Now().Unix()
		schedStart := int(now + DelayStart5)
		body := map[string]interface{}{
			"name":           schedName,
			"startSeconds":   schedStart,
			"scheduleStatus": "OS_UPDATE",
			"targetSiteId":   siteID,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			tb.Fatalf("failed marshal: %v", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, schedURL, io.NopCloser(strings.NewReader(string(bodyBytes))))
		if err != nil {
			tb.Fatalf("failed to create schedule request: %v", err)
		}
		token, err := getToken()
		if err != nil {
			tb.Fatalf("failed to get token: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := apiClient.Do(req)
		if err != nil {
			tb.Fatalf("failed to post schedule: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			tb.Fatalf("post schedule failed: %s", resp.Status)
		}
	}
}

// SetRegionSingleSched sets a single schedule for each regionID.
func SetRegionSingleSched(ctx context.Context, tb testing.TB, apiClient *http.Client, regionIDs []string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules/single", cfg.OrchFQDN, cfg.Project)
	for _, regionID := range regionIDs {
		schedName := "schedSingle"
		now := time.Now().Unix()
		schedStart := int(now + DelayStart5)
		body := map[string]interface{}{
			"name":           schedName,
			"startSeconds":   schedStart,
			"scheduleStatus": "OS_UPDATE",
			"targetRegionId": regionID,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			tb.Fatalf("failed to create schedule request: %v", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, schedURL, io.NopCloser(strings.NewReader(string(bodyBytes))))
		if err != nil {
			tb.Fatalf("failed to create schedule request: %v", err)
		}
		token, err := getToken()
		if err != nil {
			tb.Fatalf("failed to get token: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := apiClient.Do(req)
		if err != nil {
			tb.Fatalf("failed to post schedule: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			tb.Fatalf("post schedule failed: %s", resp.Status)
		}
	}
}

// CheckHostsMaintenance checks that all hosts are in maintenance state.
func CheckHostsMaintenance(ctx context.Context, tb testing.TB, apiClient *http.Client, hostIDs []string) {
	tb.Helper()
	for _, hostID := range hostIDs {
		AssertHostInMaintenance(ctx, tb, apiClient, hostID)
	}
}

// AssertHostInMaintenance checks if a host has a maintenance schedule.
func AssertHostInMaintenance(ctx context.Context, tb testing.TB, apiClient *http.Client, hostID string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules?hostId=%s", cfg.OrchFQDN, cfg.Project, hostID)
	found := false
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListSchedulesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.SingleSchedules) > 0 || len(ps.RepeatedSchedules) > 0 {
			found = true
		}
		return nil
	}
	err := httpGet(ctx, apiClient, schedURL, responseHooker)
	if err != nil || !found {
		tb.Fatalf("host %s not in maintenance: %v", hostID, err)
	}
}

// CheckSiteMaintenance checks if a site is in maintenance state.
func CheckSiteMaintenance(ctx context.Context, tb testing.TB, apiClient *http.Client, siteID string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules?siteId=%s", cfg.OrchFQDN, cfg.Project, siteID)
	found := false
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListSchedulesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.SingleSchedules) > 0 || len(ps.RepeatedSchedules) > 0 {
			found = true
		}
		return nil
	}
	err := httpGet(ctx, apiClient, schedURL, responseHooker)
	if err != nil || !found {
		tb.Fatalf("site %s not in maintenance: %v", siteID, err)
	}
}

// CheckRegionMaintenance checks if a region is in maintenance state.
func CheckRegionMaintenance(ctx context.Context, tb testing.TB, apiClient *http.Client, regionID string) {
	tb.Helper()
	cfg := flags_test.GetConfig()
	schedURL := fmt.Sprintf("https://api.%s/v1/projects/%s/schedules?regionId=%s", cfg.OrchFQDN, cfg.Project, regionID)
	found := false
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListSchedulesResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.SingleSchedules) > 0 || len(ps.RepeatedSchedules) > 0 {
			found = true
		}
		return nil
	}
	err := httpGet(ctx, apiClient, schedURL, responseHooker)
	if err != nil || !found {
		tb.Fatalf("region %s not in maintenance: %v", regionID, err)
	}
}

// ConfigureHostsbyID assigns a list of hosts to a site.
func ConfigureHostsbyID(ctx context.Context, apiClient *http.Client, hostIDs []string, siteID string) error {
	fmtHostsURL := fmt.Sprintf(hostsURL, flags_test.GetConfig().OrchFQDN, flags_test.GetConfig().Project)
	for _, hostID := range hostIDs {
		patchURL := fmt.Sprintf("%s/%s", fmtHostsURL, hostID)
		body := edgeinfraapi.HostResource{
			SiteId: &siteID, // Assign the site by setting SiteId
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, patchURL,
			io.NopCloser(strings.NewReader(string(bodyBytes))))
		if err != nil {
			return err
		}
		token, err := getToken()
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := apiClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("patch host failed: %s", resp.Status)
		}
	}
	return nil
}

// UnconfigureAllHosts unassigns all hosts from their sites.
func UnconfigureAllHosts(ctx context.Context, apiClient *http.Client) error {
	filter := "has(site)"
	hostsList, err := ListHosts(ctx, apiClient, &filter)
	if err != nil {
		return err
	}
	fmtHostsURL := fmt.Sprintf(hostsURL, flags_test.GetConfig().OrchFQDN, flags_test.GetConfig().Project)
	for hostID := range hostsList {
		patchURL := fmt.Sprintf("%s/%s", fmtHostsURL, hostID)
		body := edgeinfraapi.HostResource{
			SiteId: nil, // Unassign the site by setting SiteId to nil
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, patchURL,
			io.NopCloser(strings.NewReader(string(bodyBytes))))
		if err != nil {
			return err
		}
		token, err := getToken()
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := apiClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("patch host failed: %s", resp.Status)
		}
	}
	return nil
}

// CreateRegionAPI creates a new region in the Infrastructure Manager.
func CreateRegionAPI(ctx context.Context,
	apiClient *http.Client,
	cfg *flags_test.TestConfig,
	body *edgeinfraapi.RegionResource,
) (string, error) {
	zlog.Info().Msg("CreateRegionAPI")
	regionURL := fmt.Sprintf("https://api.%s/v1/projects/%s/regions", cfg.OrchFQDN, cfg.Project)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, regionURL,
		io.NopCloser(strings.NewReader(string(bodyBytes))))
	if err != nil {
		return "", err
	}
	token, err := getToken()
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create region failed: %s", resp.Status)
	}

	// Extract the region ID from the response
	var regionID string
	if responseHooker := func(res *http.Response) error {
		b, errRes := io.ReadAll(res.Body)
		if errRes != nil {
			return errRes
		}
		ps := &edgeinfraapi.CreateRegionResponse{}
		errMarsh := json.Unmarshal(b, &ps)
		if errMarsh != nil {
			return errMarsh
		}
		regionID = *ps.Region.ResourceId
		return nil
	}; responseHooker(resp) != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return regionID, nil
}

// CreateSiteAPI creates a new site in the Infrastructure Manager.
func CreateSiteAPI(ctx context.Context,
	apiClient *http.Client,
	cfg *flags_test.TestConfig,
	body *edgeinfraapi.SiteResource,
) (string, error) {
	zlog.Info().Msg("CreateSiteAPI")
	siteURL := fmt.Sprintf("https://api.%s/v1/projects/regions/region-12345678/%s/sites", cfg.OrchFQDN, cfg.Project)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, siteURL, io.NopCloser(strings.NewReader(string(bodyBytes))))
	if err != nil {
		return "", err
	}
	token, err := getToken()
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create site failed: %s", resp.Status)
	}

	// Extract the site ID from the response
	var siteID string
	if responseHooker := func(res *http.Response) error {
		b, errRead := io.ReadAll(res.Body)
		if errRead != nil {
			return errRead
		}
		ps := &edgeinfraapi.CreateSiteResponse{}
		errMarsh := json.Unmarshal(b, &ps)
		if errMarsh != nil {
			return errMarsh
		}
		siteID = *ps.Site.ResourceId
		return nil
	}; responseHooker(resp) != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return siteID, nil
}
