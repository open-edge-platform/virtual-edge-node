// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	edgeinfraapi "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
)

const (
	// wait time for HTTP request.
	waitTime = 60 * time.Second
)

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

	allListed := false
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.HostsList{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if ps.Hosts == nil || len(*ps.Hosts) == 0 {
			return nil
		}
		for _, host := range *ps.Hosts {
			hostsList[*host.ResourceId] = host.Uuid.String()
		}

		if !*ps.HasNext {
			zlog.Info().Msgf("All listed %v", hostsList)
			allListed = true
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
		if err := httpGet(rCtx, apiClient, fmtURL, responseHooker); err != nil {
			cancel()
			return hostsList, err
		}
		cancel()

		if allListed {
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

	allListed := false
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.InstanceList{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if ps.Instances == nil || len(*ps.Instances) == 0 {
			return nil
		}
		for _, res := range *ps.Instances {
			resList[*res.ResourceId] = *res.Host.ResourceId
		}

		if !*ps.HasNext {
			zlog.Info().Msgf("All listed %v", resList)
			allListed = true
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
		if err := httpGet(rCtx, apiClient, fmtURL, responseHooker); err != nil {
			cancel()
			return resList, err
		}
		cancel()

		if allListed {
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
