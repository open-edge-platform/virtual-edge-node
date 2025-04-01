// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/magefile/mage/sh"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	clientTokenPerm = 0o644
)

var zlog = logging.GetLogger("keycloak")

func GetKeycloakToken(
	ctx context.Context,
	certCA, orchFQDN, keycloakUser, keycloakPassword, clientID string,
) (string, error) {
	zlog.Debug().Msg("Get Keycloak API Token")
	URL := fmt.Sprintf("https://keycloak.%s/realms/master/protocol/openid-connect/token", orchFQDN)
	zlog.Debug().Msgf("Fetching Access Token from %s", URL)

	data := url.Values{}
	data.Set("username", keycloakUser)
	data.Set("password", keycloakPassword)
	data.Add("grant_type", "password")
	data.Add("client_id", clientID)
	data.Add("scope", "openid")
	encodedData := data.Encode()

	bodyReader := strings.NewReader(encodedData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

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
	defer res.Body.Close()

	zlog.Debug().Msgf("client: got response status code: %d", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}
	zlog.Debug().Msgf("client: response body: %s", resBody)

	replyBody := map[string]interface{}{}
	err = json.Unmarshal(resBody, &replyBody)
	if err != nil {
		zlog.Err(err).Msg("failed to umarshal response body")
		return "", err
	}

	token, ok := replyBody["access_token"]
	if !ok {
		err = fmt.Errorf("could not get access token from data %s", replyBody)
		zlog.Err(err).Msg("failed to get token")
		return "", err
	}
	tokenStr, ok := token.(string)
	if !ok {
		err = fmt.Errorf("could not get access token from data %s", replyBody)
		zlog.Err(err).Msg("failed to get token")
		return "", err
	}
	zlog.Debug().Msgf("Retrieved Keycloak Access Token: %s", tokenStr)
	return tokenStr, nil
}

func getKeycloakClientToken(ctx context.Context, certCA, orchFQDN, clientName, clientSecret string) (string, error) {
	zlog.Debug().Msgf("Get Keycloak Client Token clientName, clientSecret %s, %s", clientName, clientSecret)
	URL := fmt.Sprintf("https://keycloak.%s/realms/master/protocol/openid-connect/token", orchFQDN)
	zlog.Debug().Msgf("Fetching Client Token from %s", URL)

	data := url.Values{}
	data.Add("grant_type", "client_credentials")
	data.Set("client_id", clientName)
	data.Set("client_secret", clientSecret)
	encodedData := data.Encode()

	bodyReader := strings.NewReader(encodedData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

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
	defer res.Body.Close()

	zlog.Debug().Msgf("client: got response status code: %d", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}
	zlog.Debug().Msgf("client: response body: %s", resBody)

	replyBody := map[string]interface{}{}
	err = json.Unmarshal(resBody, &replyBody)
	if err != nil {
		zlog.Err(err).Msg("failed to umarshal response body")
		return "", err
	}

	clientToken, ok := replyBody["access_token"]
	if !ok {
		err = fmt.Errorf("could not get access token from data %s", replyBody)
		zlog.Err(err).Msg("failed to get token")
		return "", err
	}
	tokenStr, ok := clientToken.(string)
	if !ok {
		err = fmt.Errorf("could not get access token from data %s", replyBody)
		zlog.Err(err).Msg("failed to get token")
		return "", err
	}

	zlog.Debug().Msgf("Retrieved Keycloak Client Token: %s", tokenStr)
	return tokenStr, nil
}

func Folders(settings *defs.Settings) error {
	// Creates folders for credentials / tokens
	credentialsFolder := settings.BaseFolder + defs.ENClientFolder
	zlog.Info().Msgf("Creating Keycloack folders %s and %v", credentialsFolder, defs.TokenFolders)

	err := sh.RunV("mkdir", "-p", credentialsFolder)
	if err != nil {
		zlog.Err(err).Msgf("failed to create %s", credentialsFolder)
		return err
	}
	for _, tkFolder := range defs.TokenFolders {
		errV := sh.RunV("mkdir", "-p", settings.BaseFolder+tkFolder)
		if errV != nil {
			zlog.Err(errV).Msgf("failed to create %s", tkFolder)
			return errV
		}
	}
	return nil
}

func SouthKeycloack(ctx context.Context, settings *defs.Settings) error {
	zlog.Info().Msgf("Setting Keycloack Artifacts for Edge Node %s", settings.ENGUID)

	clientID, err := utils.LoadFile(settings.BaseFolder + defs.ENClientIDPath)
	if err != nil {
		zlog.Err(err).Msgf("failed to read client ID file %s", settings.BaseFolder+defs.ENClientIDPath)
		return err
	}

	clientSecret, err := utils.LoadFile(settings.BaseFolder + defs.ENClientSecretPath)
	if err != nil {
		zlog.Err(err).Msgf("failed to read client secret file %s", settings.BaseFolder+defs.ENClientSecretPath)
		return err
	}

	clientID = strings.Trim(clientID, "\n")
	clientSecret = strings.Trim(clientSecret, "\n")
	clientToken, err := getKeycloakClientToken(ctx, settings.CertCA, settings.OrchFQDN, clientID, clientSecret)
	if err != nil {
		zlog.Err(err).Msgf("failed to get client token")
		return err
	}

	err = os.WriteFile(settings.BaseFolder+defs.ENClientTokenPath, []byte(clientToken), clientTokenPerm)
	if err != nil {
		zlog.Err(err).Msgf("could not write client token file")
		return err
	}

	zlog.Info().Msgf("Finished Keycloack configs %s", settings.ENGUID)
	return nil
}
