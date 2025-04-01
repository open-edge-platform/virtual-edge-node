// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	TIMEOUT     = 30 * time.Second
	EARLYEXPIRY = 15 * time.Minute
)

type Client struct {
	HostGUID   string
	BaseURL    *url.URL
	HTTPClient *http.Client
}

type CSRPayload struct {
	Csr []byte `json:"csr"`
}

type CertResponseContent struct {
	Cert string `json:"certificate"`
}

type CertResponsePayload struct {
	Success CertResponseContent `json:"success"`
}

func newAuth(serverURL, hostGUID string, tlsConfig *tls.Config, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil || u == nil {
		return nil, fmt.Errorf("url is not valid : %w", err)
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	client := Client{
		HostGUID:   hostGUID,
		BaseURL:    u,
		HTTPClient: &http.Client{Transport: tr, Timeout: timeout},
	}

	return &client, nil
}

func GetAuthCli(idpURL, guid string, caCertPool *x509.CertPool) (*Client, error) {
	baseEndpoint := fmt.Sprintf("https://%s", idpURL)
	tlsConfig := tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}
	authCli, err := newAuth(baseEndpoint, guid, &tlsConfig, TIMEOUT)
	return authCli, err
}

func GetAccessTokenFromResponse(response *http.Response) (string, error) {
	if response.StatusCode < 200 || response.StatusCode > 299 {
		err := fmt.Errorf("failed response status code %s", response.Status)
		return "", err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read resp.Body:%w", err)
	}
	var tokenR map[string]interface{}
	err = json.Unmarshal(bodyBytes, &tokenR)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response:%w", err)
	}

	at, exists := tokenR[AccessToken].(string)
	if !exists {
		return "", fmt.Errorf("failed to parse token response")
	}
	return at, nil
}

func (cli *Client) ProvisionAccessToken(ctx context.Context, authConf ConfigAuth) (oauth2.Token, error) {
	var token oauth2.Token
	clientIDFile := filepath.Join(authConf.ClientCredsPath, "client_id")
	clientID, err := ReadFileNoLinks(clientIDFile)
	if err != nil {
		return token, fmt.Errorf("failed to read client id file: %w", err)
	}
	clientSecretFile := filepath.Join(authConf.ClientCredsPath, "client_secret")
	secret, err := ReadFileNoLinks(clientSecretFile)
	if err != nil {
		return token, fmt.Errorf("failed to read client secret file: %w", err)
	}
	endpoint := cli.BaseURL.JoinPath("/realms/master/protocol/openid-connect/token")

	payload := url.Values{}
	payload.Set("grant_type", "client_credentials")
	payload.Set("client_id", strings.TrimSpace(string(clientID)))
	payload.Set("client_secret", strings.TrimSpace(string(secret)))

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader([]byte(payload.Encode())),
	)
	if err != nil {
		return token, fmt.Errorf("http request creation failed: %w", err)
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := cli.HTTPClient.Do(request)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get token from IDP service")
		return token, err
	}
	defer response.Body.Close()

	at, err := GetAccessTokenFromResponse(response)
	if err != nil {
		return token, fmt.Errorf("failed to get access token from response:%w", err)
	}

	exp, err := GetExpiryFromJWT(at)
	if err != nil {
		return token, fmt.Errorf("failed to get expiry from token:%w", err)
	}

	zlog.Debug().Msgf("token retrieved from IDP successfully")
	return oauth2.Token{AccessToken: at, Expiry: exp}, nil
}

func (cli *Client) ProvisionReleaseServiceToken(
	ctx context.Context,
	accessToken string,
) (oauth2.Token, error) {
	var token oauth2.Token
	endpoint := cli.BaseURL.JoinPath("/token")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return token, fmt.Errorf("http request creation failed: %w", err)
	}

	// Add access token in request header
	bearer := "Bearer " + accessToken

	request.Header.Add("Authorization", bearer)
	response, err := cli.HTTPClient.Do(request)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get release service token")
		return token, err
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return token, fmt.Errorf("failed to read resp.Body:%w", err)
	}
	relToken := string(bodyBytes)

	zlog.Debug().Msgf("release service token retrieved successfully")
	expiry, err := GetExpiryFromJWT(strings.Trim(relToken, "\""))
	if err != nil {
		return token, fmt.Errorf("failed to parse jwt release token to get expiry :%w", err)
	}
	return oauth2.Token{AccessToken: relToken, Expiry: expiry}, nil
}

func (cli *Client) ProvisionToken(
	ctx context.Context,
	authConf ConfigAuth,
	tknClient ClientAuthToken,
) (oauth2.Token, error) {
	var tokenContent string
	var token oauth2.Token
	var err error
	// provision release service token
	if tknClient.ClientName == "release-service" {
		tokenContent, err = GetNodeAgentToken(authConf)
		if err != nil {
			zlog.Error().Err(err).Msgf("failed to load release service token")
			return token, err
		}
		token, err = cli.ProvisionReleaseServiceToken(ctx, tokenContent)
		if err != nil {
			zlog.Error().Err(err).Msgf("failed to get release service token")
			return token, err
		}
	} else {
		// provision orch service token
		token, err = cli.ProvisionAccessToken(ctx, authConf)
		if err != nil {
			zlog.Error().Err(err).Msgf("failed to get access token from IDP")
			return token, err
		}
	}

	zlog.Debug().Msgf("JWT token generated for client %s", tknClient.ClientName)
	errPT := PersistToken(token.AccessToken, filepath.Join(authConf.AccessTokenPath, tknClient.ClientName, AccessToken))
	if errPT != nil {
		zlog.Error().Err(errPT).Msgf("failed to Persist token to file")
		return token, errPT
	}
	return token, nil
}
