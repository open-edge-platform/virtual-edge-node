// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

func CreateOrg(ctx context.Context, certCA, orchFQDN, apiToken, orgName string) (string, error) {
	zlog.Debug().Msgf("CreateOrg %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s", orchFQDN, orgName)
	zlog.Debug().Msgf("CreateOrg from %s", URL)

	body := map[string]interface{}{
		"description": orgName,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}

	orgStatus := string(resBody)
	zlog.Debug().Msgf("Created Org: %s", orgStatus)
	return orgStatus, nil
}

func GetOrg(ctx context.Context, certCA, orchFQDN, apiToken, orgName string) (string, error) {
	zlog.Debug().Msgf("CreateOrg %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s", orchFQDN, orgName)
	zlog.Debug().Msgf("CreateOrg from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}

	orgStatus := string(resBody)
	zlog.Debug().Msgf("Got Org: %s", orgStatus)
	return orgStatus, nil
}

func GetOrgLicenses(ctx context.Context, certCA, orchFQDN, apiToken, orgName string) (string, error) {
	zlog.Debug().Msgf("GetOrgLicenses %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s/licenses", orchFQDN, orgName)
	zlog.Debug().Msgf("GetOrgLicenses from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}

	orgStatus := string(resBody)
	zlog.Debug().Msgf("Got OrgLicenses: %s", orgStatus)
	return orgStatus, nil
}

func CreateOrgLicense(ctx context.Context, certCA, orchFQDN, apiToken, orgName, customerID, productKey string) error {
	zlog.Debug().Msgf("CreateOrgLicense %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s/licenses/enlicense", orchFQDN, orgName)
	zlog.Debug().Msgf("CreateOrgLicense from %s", URL)

	body := map[string]interface{}{
		"customerID": customerID,
		"productKey": productKey,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, URL, bodyReader)
	if err != nil {
		return err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return err
	}

	replyBody := map[string]interface{}{}
	err = json.Unmarshal(resBody, &replyBody)
	if err != nil {
		zlog.Err(err).Msg("failed to umarshal response body")
		return err
	}

	zlog.Debug().Msgf("Created Org License: %v", replyBody)
	return nil
}

func DeleteOrgLicense(ctx context.Context, certCA, orchFQDN, apiToken, orgName string) error {
	zlog.Debug().Msgf("DeleteOrgLicense %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s/licenses/enlicense", orchFQDN, orgName)
	zlog.Debug().Msgf("DeleteOrgLicense from %s", URL)

	body := map[string]interface{}{}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, URL, bodyReader)
	if err != nil {
		return err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return err
	}

	replyBody := map[string]interface{}{}
	err = json.Unmarshal(resBody, &replyBody)
	if err != nil {
		zlog.Err(err).Msg("failed to umarshal response body")
		return err
	}

	zlog.Debug().Msgf("DeleteOrgLicense: %v", replyBody)
	return nil
}

func CreateProject(ctx context.Context, certCA, orchFQDN, apiToken, projName string) (string, error) {
	zlog.Debug().Msgf("CreateProject %s", projName)
	URL := fmt.Sprintf("https://api.%s/v1/projects/%s", orchFQDN, projName)
	zlog.Debug().Msgf("CreateProject from %s", URL)

	body := map[string]interface{}{
		"description": projName,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}

	projStatus := string(resBody)
	zlog.Debug().Msgf("Created Project: %s", projStatus)
	return projStatus, nil
}

func GetProject(ctx context.Context, certCA, orchFQDN, apiToken, projName string) (string, error) {
	zlog.Debug().Msgf("GetProject %s", projName)
	URL := fmt.Sprintf("https://api.%s/v1/projects/%s", orchFQDN, projName)
	zlog.Debug().Msgf("GetProject from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}

	projStatus := string(resBody)
	zlog.Debug().Msgf("Got Project: %s", projStatus)
	return projStatus, nil
}

func DeleteProject(ctx context.Context, certCA, orchFQDN, apiToken, projName string) error {
	zlog.Debug().Msgf("DeleteProject %s", projName)
	URL := fmt.Sprintf("https://api.%s/v1/projects/%s", orchFQDN, projName)
	zlog.Debug().Msgf("DeleteProject from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, URL, bodyReader)
	if err != nil {
		return err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return err
	}
	defer res.Body.Close()

	zlog.Debug().Msgf("Deleted Project: %s", projName)
	return nil
}

func DeleteOrg(ctx context.Context, certCA, orchFQDN, apiToken, orgName string) error {
	zlog.Debug().Msgf("DeleteOrg %s", orgName)
	URL := fmt.Sprintf("https://api.%s/v1/orgs/%s", orchFQDN, orgName)
	zlog.Debug().Msgf("DeleteOrg from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, URL, bodyReader)
	if err != nil {
		return err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return err
	}
	defer res.Body.Close()

	zlog.Debug().Msgf("Deleted Org: %s", orgName)
	return nil
}

func ListProjects(ctx context.Context, certCA, orchFQDN, apiToken string) (string, error) {
	zlog.Debug().Msgf("ListProjects")
	URL := fmt.Sprintf("https://api.%s/v1/projects", orchFQDN)
	zlog.Debug().Msgf("ListProjects from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}
	projects := string(resBody)
	zlog.Debug().Msgf("Got Projects: %s", projects)
	return projects, nil
}

func ListOrgs(ctx context.Context, certCA, orchFQDN, apiToken string) (string, error) {
	zlog.Debug().Msgf("ListOrgs")
	URL := fmt.Sprintf("https://api.%s/v1/orgs", orchFQDN)
	zlog.Debug().Msgf("ListOrgs from %s", URL)

	body := map[string]interface{}{}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	bodyReader := bytes.NewReader(bodyJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL, bodyReader)
	if err != nil {
		return "", err
	}

	authHeader := fmt.Sprintf("Bearer %s", apiToken)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", authHeader)

	client, err := utils.GetClientWithCA(certCA)
	if err != nil {
		zlog.Err(err).Msgf("failed to get http client")
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		zlog.Err(err).Msgf("client: error making http request")
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = fmt.Errorf("error client status code %s", res.Status)
		zlog.Err(err).Msgf("failed call")
		return "", err
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}
	orgs := string(resBody)
	zlog.Debug().Msgf("Got Orgs: %s", orgs)
	return orgs, nil
}
