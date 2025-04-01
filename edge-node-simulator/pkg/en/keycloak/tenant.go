// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tenantIDRoleSeparator = "_"
	uuidPattern           = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	tenantIDPath          = "/tenantId"
	tenantPerm            = 0o640
)

var uuidRegex = regexp.MustCompile(uuidPattern)

type TokenClaims struct {
	jwt.RegisteredClaims
	RealmAccess RealmAccess `json:"realm_access"`
}

type RealmAccess struct {
	Roles []string `json:"roles"`
}

// Creates tenant ID file, if it already exists returns nil.
func CreateTenantID(confAuth *ConfigAuth, token string) error {
	tenantIDFilepath := filepath.Join(confAuth.ClientCredsPath, tenantIDPath)
	return createTenantID(tenantIDFilepath, token)
}

func createTenantID(path, token string) error {
	file, err := CreateExcl(path, tenantPerm)
	if errors.Is(err, os.ErrExist) {
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()

	tenantID, err := getTenantID(token)
	if err != nil {
		removeFile(file.Name())
		return err
	}

	_, err = file.WriteString("TENANT_ID=" + tenantID)
	if err != nil {
		removeFile(file.Name())
		return err
	}

	return nil
}

func getTenantID(token string) (string, error) {
	parser := &jwt.Parser{}
	t, _, err := parser.ParseUnverified(token, &TokenClaims{})
	if err != nil {
		return "", err
	}

	claims, ok := t.Claims.(*TokenClaims)
	if !ok {
		return "", fmt.Errorf("unknown claims type")
	}

	var tenantIDs []string
	for _, role := range claims.RealmAccess.Roles {
		if strings.Contains(role, tenantIDRoleSeparator) {
			roleTID := strings.Split(role, tenantIDRoleSeparator)[0]
			if !uuidRegex.MatchString(roleTID) {
				continue
			}

			if !slices.Contains(tenantIDs, roleTID) {
				tenantIDs = append(tenantIDs, roleTID)
			}
		}
	}

	if len(tenantIDs) == 0 {
		return "", fmt.Errorf("no tenant ID found in JWT")
	}
	if len(tenantIDs) > 1 {
		return "", fmt.Errorf("multiple tenant IDs found in JWT: %v", tenantIDs)
	}
	return tenantIDs[0], nil
}

func removeFile(path string) {
	err := os.Remove(path)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to remove %v", path)
	}
}
