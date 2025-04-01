// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/Nerzal/gocloak/v13"
	"google.golang.org/grpc/codes"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

type KeycloakService struct {
	keycloakClient *gocloak.GoCloak
	jwtToken       *gocloak.JWT
}

func NewKeycloakService(ctx context.Context,
	keycloakURL, clientName, clientSecret, realm, caCertPath string,
) (*KeycloakService, error) {
	kss := &KeycloakService{}

	err := kss.login(ctx, keycloakURL, clientName, clientSecret, realm, caCertPath)
	if err != nil {
		return nil, err
	}

	return kss, nil
}

func (k *KeycloakService) login(ctx context.Context,
	keycloakURL, clientName, clientSecret, realm, caCertPath string,
) error {
	client := gocloak.NewClient(keycloakURL)

	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read CA certificate %s", caCertPath)
			zlog.Error().Err(err).Msg(errMsg)
			return fmt.Errorf("%s", errMsg)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		client.RestyClient().SetTLSClientConfig(&tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		})
	}

	jwtToken, err := client.LoginAdmin(ctx, clientName, clientSecret, realm)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to login to Keycloak %s", keycloakURL)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	k.keycloakClient = client
	k.jwtToken = jwtToken

	zlog.InfraSec().Debug().Msgf("Keycloak client logged in successfully")
	return nil
}

func (k *KeycloakService) Logout(ctx context.Context, kcUser, kcRealm string) error {
	// refresh_token is required to logout but it's not provided for all Keycloak clients.
	// Skip logging out if refresh_token is not provided.
	if k.jwtToken.RefreshToken == "" {
		return nil
	}
	if err := k.keycloakClient.Logout(ctx, kcUser, kcRealm, k.jwtToken.AccessToken, k.jwtToken.RefreshToken); err != nil {
		zlog.InfraSec().Err(err).Msgf("Failed to logout from Keycloak")
		return err
	}
	return nil
}

func (k *KeycloakService) GetToken(ctx context.Context, realm, username, password string) (string, error) {
	if k.jwtToken.AccessToken == "" {
		return "", inv_errors.Errorf("access token is not available")
	}

	clientID := "system-client"
	scopes := []string{"openid", "profile", "email", "groups"}
	options := gocloak.TokenOptions{
		Scopes:    &scopes,
		Username:  &username,
		Password:  &password,
		ClientID:  &clientID,
		GrantType: gocloak.StringP("password"),
	}
	jwt, err := k.keycloakClient.GetToken(ctx, realm, options)
	if err != nil {
		zlog.InfraSec().Err(err).Msg("failed to get token")
		return "", err
	}

	return jwt.AccessToken, nil
}

func (k *KeycloakService) CreateUser(ctx context.Context,
	keycloakRealm, username, password string,
) (string, error) {
	zlog.Debug().Msgf("Creating Keycloak user ID for username %s", username)
	user := gocloak.User{
		Username:      &username,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
		Credentials: &[]gocloak.CredentialRepresentation{
			{
				Type:      gocloak.StringP("password"),
				Value:     &password,
				Temporary: gocloak.BoolP(false),
			},
		},
	}

	userID, err := k.keycloakClient.CreateUser(ctx, k.jwtToken.AccessToken, keycloakRealm, user)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create Keycloak user %s", *user.Username)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}
	zlog.Debug().Msgf("Created Keycloak userID %s for username %s", userID, username)
	return userID, nil
}

func (k *KeycloakService) GetUserIDByName(ctx context.Context,
	realm, username string,
) (string, error) {
	zlog.Debug().Msgf("Getting Keycloak user ID for username %s", username)

	svcAccountUsers, err := k.keycloakClient.GetUsers(ctx, k.jwtToken.AccessToken, realm, gocloak.GetUsersParams{
		Username: &username,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Cannot retrieve Keycloak user %s", username)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}

	if len(svcAccountUsers) == 0 {
		invErr := inv_errors.Errorfc(
			codes.NotFound, "No Keycloak user found with username %s", username)
		zlog.InfraSec().Err(invErr).Msg("")
		return "", invErr
	}

	// This should never happen but we could have more than one Keycloak user with the same username.
	// We print warning and get first.
	if len(svcAccountUsers) > 1 {
		zlog.Warn().Msgf(
			"More than one Keycloak user found for username %s, getting first one", username)
	}

	svcAccountUserID := *svcAccountUsers[0].ID
	zlog.Debug().Msgf("Got Keycloak user ID %s for username %s", svcAccountUserID, username)
	return svcAccountUserID, nil
}

func (k *KeycloakService) AddUserToGroup(ctx context.Context,
	realm, username, groupname string,
) error {
	zlog.Debug().Msgf("Adding user to group [user=%s, group=%s]",
		username, groupname)

	groupID, err := k.GetGroupIDByName(ctx, realm, groupname)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot get group %s",
			groupname)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}
	userID, err := k.GetUserIDByName(ctx, realm, username)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot get user %s",
			username)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	err = k.keycloakClient.AddUserToGroup(ctx, k.jwtToken.AccessToken, realm, userID, groupID)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot add group %s to user %s",
			userID, groupID)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	zlog.Debug().Msgf("Added user to group [user=%s, group=%s]",
		username, groupname)
	return nil
}

func (k *KeycloakService) GetGroupIDByName(ctx context.Context,
	realm, groupName string,
) (string, error) {
	groups, err := k.keycloakClient.GetGroups(ctx, k.jwtToken.AccessToken, realm, gocloak.GetGroupsParams{
		Search: &groupName,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak group %s", groupName)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}

	if len(groups) == 0 {
		errMsg := fmt.Sprintf("No Keycloak group found for %s", groupName)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorfc(codes.NotFound, "%s", errMsg)
	}

	// This should never happen but we could have more than one group with the same name.
	// We print warning and get first.
	if len(groups) > 1 {
		zlog.Warn().Msgf("More than one Keycloak group found for %s, getting first one", groupName)
	}
	groupID := *groups[0].ID

	return groupID, nil
}

func (k *KeycloakService) DeleteUser(ctx context.Context,
	realm, username string,
) error {
	zlog.Debug().Msgf("Deleting user [user=%s]",
		username)

	userID, err := k.GetUserIDByName(ctx, realm, username)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot get user %s",
			username)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	err = k.keycloakClient.DeleteUser(ctx, k.jwtToken.AccessToken, realm, userID)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot delete user %s",
			userID)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	zlog.Debug().Msgf("Deleted user [user=%s]", username)
	return nil
}
