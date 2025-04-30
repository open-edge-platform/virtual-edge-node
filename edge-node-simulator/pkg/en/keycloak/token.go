// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const (
	refreshInterval           = 10 * time.Minute
	refreshCheckInterval      = 600 * time.Second
	tokenRefreshCheckInterval = 300 * time.Second
)

type ClientAuthToken struct {
	ClientName  string
	AccessToken string
	Expiry      time.Time
}

type TokenManager struct {
	TokenClients []ClientAuthToken
	AuthCfg      ConfigAuth
}

const AccessToken = "access_token"

type ConfigAuth struct {
	UUID            string
	AccessTokenURL  string
	RsTokenURL      string
	AccessTokenPath string
	ClientCredsPath string
	TokenClients    []string
	CertCA          string
}

func PersistToken(token, tokenFile string) error {
	// Only the node-agent user should be able
	// to access then token

	err := os.WriteFile(tokenFile, []byte(token), clientTokenPerm) // #nosec G306
	if err != nil {
		zlog.Error().Msgf("could not persist token")
		return err
	}

	zlog.Debug().Msgf("persisted token to file")

	return nil
}

func IsTokenRefreshRequired(tokenExpiry time.Time) bool {
	safeInterval := tokenExpiry.Add(-refreshInterval)
	return time.Now().After(safeInterval)
}

func NewTokenManager(conf ConfigAuth) *TokenManager {
	tknMgr := TokenManager{
		AuthCfg:      conf,
		TokenClients: make([]ClientAuthToken, len(conf.TokenClients)),
	}

	for i := 0; i < len(conf.TokenClients); i++ {
		tknMgr.TokenClients[i] = ClientAuthToken{ClientName: conf.TokenClients[i]}
	}
	return &tknMgr
}

func (tknMgr *TokenManager) PopulateTokenClients(conf ConfigAuth) error {
	for i, client := range tknMgr.TokenClients {
		tPath := filepath.Join(conf.AccessTokenPath, client.ClientName, AccessToken)
		zlog.Debug().Msgf("Populate Token Clients path %s", tPath)
		tokenData, err := ReadFileNoLinks(tPath)
		if err != nil {
			zlog.Error().Err(err).Msgf("failed to read persistent JWT token")
			continue
		}
		tknMgr.TokenClients[i].AccessToken = string(tokenData)
		expiry, err := GetExpiryFromJWT(string(tokenData))
		if err != nil {
			zlog.Error().Err(err).Msgf("Failed to get expiry from JWT token")
			return err
		}
		tknMgr.TokenClients[i].Expiry = expiry
	}
	return nil
}

func (tknMgr *TokenManager) Start(termChan chan bool, wg *sync.WaitGroup) error {
	zlog.Info().Msgf("Token Manager - Starting refresh token routine %s", tknMgr.AuthCfg.UUID)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(tknMgr.AuthCfg.CertCA))

	authCli, err := GetAuthCli(tknMgr.AuthCfg.AccessTokenURL, tknMgr.AuthCfg.UUID, caCertPool)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to create IDP client %s", tknMgr.AuthCfg.UUID)
		return err
	}

	// Populate all clients with JWT if already provisioned
	err = tknMgr.PopulateTokenClients(tknMgr.AuthCfg)
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to Populate Token Clients: %s", tknMgr.AuthCfg.UUID)
		return err
	}
	// Initialize token for all configured clients
	createRefreshTokens(tknMgr, &tknMgr.AuthCfg, authCli)

	wg.Add(1)
	// Go-routine to manage JWT token lifecycle for NA and other Agents
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(tokenRefreshCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-termChan:
				zlog.Info().Msgf("Token Manager - Terminating refresh token routine %s", tknMgr.AuthCfg.UUID)
				return
			case <-ticker.C:
				// Renew token for all configured clients
				createRefreshTokens(tknMgr, &tknMgr.AuthCfg, authCli)
			}
		}
	}()

	return nil
}

func createRefreshTokens(tokMgr *TokenManager, confAuth *ConfigAuth, authCli *Client) {
	ctx := context.Background()
	for i, tokenClient := range tokMgr.TokenClients {
		var err error
		var token oauth2.Token
		// If token is already provisioned, manage lifecycle
		if tokenClient.AccessToken != "" {
			if IsTokenRefreshRequired(tokenClient.Expiry) {
				token, err = authCli.ProvisionToken(ctx, *confAuth, tokenClient)
				if err != nil {
					zlog.Error().Err(err).Msgf("failed to manage token")
					continue
				}
				tokMgr.TokenClients[i].AccessToken = token.AccessToken
				tokMgr.TokenClients[i].Expiry = token.Expiry
				zlog.Debug().
					Msgf("JWT token refreshed for client %s successfully %s",
						tokenClient.ClientName, tokMgr.AuthCfg.UUID)
			}
		} else {
			// provision release service token
			token, err = authCli.ProvisionToken(ctx, *confAuth, tokenClient)
			if err != nil {
				zlog.Error().Err(err).Msgf("Failed to manage token")
				continue
			}
			tokMgr.TokenClients[i].AccessToken = token.AccessToken
			tokMgr.TokenClients[i].Expiry = token.Expiry
			zlog.Info().Msgf("JWT token freshly provisioned for client %s successfully %s",
				tokenClient.ClientName, tokMgr.AuthCfg.UUID)
		}

		if tokenClient.ClientName == "node-agent" {
			err = CreateTenantID(confAuth, tokMgr.TokenClients[i].AccessToken)
			if err != nil {
				zlog.Error().Err(err).Msgf("Failed to create tenant ID file")
			}
		}
	}
}

func GetExpiryFromJWT(jwtTokenStr string) (time.Time, error) {
	var exp time.Time
	parser := &jwt.Parser{}
	token, _, err := parser.ParseUnverified(jwtTokenStr, jwt.MapClaims{})
	if err != nil {
		zlog.Error().Err(err).Msg("failed to parse jwt MapClaims from token")
		return exp, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		zlog.Error().Err(err).Msg("failed to get jwt MapClaims from token")
		return exp, err
	}
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		errParse := fmt.Errorf("failed to parse float64 exp value")
		zlog.Error().Err(errParse).Msg("failed to get exp from jwt MapClaims")
		return exp, err
	}
	exp = time.Unix(int64(expFloat), 0)

	return exp, nil
}

func GetNodeAgentToken(confs ConfigAuth) (string, error) {
	tokenFile := filepath.Join(confs.AccessTokenPath, "node-agent", AccessToken)
	tBytes, err := ReadFileNoLinks(tokenFile)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to read file node-agent token")
		return "", err
	}
	return strings.TrimSpace(string(tBytes)), nil
}

func OpenNoLinks(path string) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDONLY, 0)
}

func CreateNoLinks(path string, perm os.FileMode) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
}

func ReadFileNoLinks(path string) ([]byte, error) {
	f, err := OpenNoLinks(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func isHardLink(path string) (bool, error) {
	var stat syscall.Stat_t

	err := syscall.Stat(path, &stat)
	if err != nil {
		return false, err
	}

	if stat.Nlink > 1 {
		return true, nil
	}

	return false, nil
}

func openFileNoLinks(path string, flags int, perm os.FileMode) (*os.File, error) {
	// O_NOFOLLOW - If the trailing component (i.e., basename) of pathname is a symbolic link,
	// then the open fails, with the error ELOOP.
	file, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW, perm)
	if err != nil {
		return nil, err
	}

	hardLink, err := isHardLink(path)
	if err != nil {
		file.Close()
		return nil, err
	}

	if hardLink {
		file.Close()
		return nil, fmt.Errorf("%v is a hardlink", path)
	}

	return file, nil
}

func CreateExcl(path string, perm os.FileMode) (*os.File, error) {
	return openFileNoLinks(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
}
