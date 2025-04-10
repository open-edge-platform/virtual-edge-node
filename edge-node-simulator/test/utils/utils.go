// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("utils")

const (
	ENVJWTToken = "JWT_TOKEN"
)

var tokenRefreshInterval = 50 * time.Minute

//nolint:gosec // InsecureSkipVerify: true, as we are using CA cert
func GetHTTPClientWithCA(caCert string) (*http.Client, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert))
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return &http.Client{
		Transport: transport,
	}, nil
}

func VerifyCA(caCert, fqdn string) error {
	block, _ := pem.Decode([]byte(caCert))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		zlog.Err(err).Msgf("failed to parse CA cert")
		return err
	}

	zlog.Info().Msgf("CA cert %s", cert.PermittedURIDomains)

	_, err = cert.Verify(x509.VerifyOptions{
		DNSName:     fqdn,
		CurrentTime: time.Now(),
	})
	if err != nil {
		zlog.Err(err).Msgf("failed to verify CA cert")
		return err
	}
	return nil
}

func GetOrchAPIToken(
	ctx context.Context,
	certCA, orchFQDN, keycloakUser, keycloakPassword string,
) (string, error) {
	zlog.Info().Msg("Get Orch API Token")
	URL := fmt.Sprintf("https://keycloak.%s/realms/master/protocol/openid-connect/token", orchFQDN)

	data := url.Values{}
	data.Set("username", keycloakUser)
	data.Set("password", keycloakPassword)
	data.Add("grant_type", "password")
	data.Add("client_id", "system-client")
	data.Add("scope", "openid")
	encodedData := data.Encode()

	bodyReader := strings.NewReader(encodedData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bodyReader)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client, err := GetHTTPClientWithCA(certCA)
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

	zlog.Info().Msgf("client: got response")
	zlog.Info().Msgf("client: status code: %d", res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		zlog.Err(err).Msgf("client: could not read response body")
		return "", err
	}
	zlog.Info().Msgf("client: response body: %s", resBody)

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
		zlog.Err(err).Msg("failed to parse token into string")
		return "", err
	}
	return tokenStr, nil
}

func HelperJWTTokenRoutine(ctx context.Context,
	certCA,
	orchFQDN,
	keyCloakUser,
	keyCloakPass string,
) error {
	jwtToken, err := GetOrchAPIToken(
		ctx,
		certCA,
		orchFQDN,
		keyCloakUser,
		keyCloakPass,
	)
	if err != nil {
		return err
	}
	os.Setenv(ENVJWTToken, jwtToken)
	zlog.Info().Msg("Started helperJWTTokenRoutine")
	go func() {
		tk := time.NewTicker(tokenRefreshInterval)
		for {
			select {
			case <-tk.C:
				jwtToken, err := GetOrchAPIToken(
					ctx,
					certCA,
					orchFQDN,
					keyCloakUser,
					keyCloakPass,
				)
				if err != nil {
					zlog.Error().Err(err).Msg("failed to refresh JWT token")
				}
				os.Setenv(ENVJWTToken, jwtToken)
			case <-ctx.Done():
				zlog.Info().Msg("Finished helperJWTTokenRoutine")
				return
			}
		}
	}()
	return nil
}

func LoadFile(filePath string) (string, error) {
	dirFile, err := filepath.Abs(filePath)
	if err != nil {
		zlog.Err(err).Msgf("failed LoadFile, filepath unexistent %s", filePath)
		return "", err
	}

	dataBytes, err := os.ReadFile(dirFile)
	if err != nil {
		zlog.Err(err).Msgf("failed to read file %s", dirFile)
		return "", err
	}

	dataStr := string(dataBytes)
	return dataStr, nil
}

//nolint:gosec // InsecureSkipVerify: true, as we are using CA cert / test function
func GetClientWithCA(caCert string) (*http.Client, error) {
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		err := fmt.Errorf("failed to parse CA cert into http client")
		return nil, err
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return &http.Client{
		Transport: transport,
	}, nil
}

func CheckResponse(expected, current int) error {
	if expected != current {
		err := fmt.Errorf("")
		zlog.Error().Err(err).Msgf("failed http response: expected %d current %d", expected, current)
		return err
	}
	return nil
}

func CheckErrors(errChan chan error) error {
	allErrors := []string{}
	for errDel := range errChan {
		if errDel != nil {
			allErrors = append(allErrors, errDel.Error())
		}
	}
	if len(allErrors) > 0 {
		errAllDel := fmt.Errorf("%v", allErrors)
		return errAllDel
	}
	return nil
}
