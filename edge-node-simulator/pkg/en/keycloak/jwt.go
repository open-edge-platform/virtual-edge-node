// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// SharedSecretKey environment variable name for shared secret key for signing a token.
	SharedSecretKey = "SHARED_SECRET_KEY"
	secretKey       = "randomSecretKey"
	writeRole       = "infra-manager-core-write-role"
	readRole        = "infra-manager-core-read-role"
	enReadWriteRole = "node-agent-readwrite-role"
)

func WithTenantID(tenantID string) Option {
	return func(o *Options) {
		o.tenantIDs = append(o.tenantIDs, tenantID)
	}
}

type Options struct {
	tenantIDs []string
}

type Option func(*Options)

// parseOptions parses the given list of Option into an Options.
func parseOptions(options ...Option) *Options {
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

// CreateJWT returns random signing key and JWT token (HS256 encoded) in a string with both roles, read and write.
// Only 1 token can persist in the system (otherwise, env variable holding secret key would be re-written).
func CreateJWT() (string, string, error) {
	claims := &jwt.MapClaims{
		"iss": "https://keycloak.kind.internal/realms/master",
		"exp": time.Now().Add(time.Hour).Unix(),
		"typ": "Bearer",
		"realm_access": map[string]interface{}{
			"roles": []string{
				writeRole,
				readRole,
			},
		},
	}

	return CreateJWTWithClaims(claims)
}

// CreateENJWT returns random signing key and JWT token (HS256 encoded) in a string with EN's read-write role.
// Only 1 token can persist in the system (otherwise, env variable holding secret key would be re-written).
func CreateENJWT(opts ...Option) (string, string, error) {
	options := parseOptions(opts...)
	roles := []string{
		"default-roles-master",
		"release-service-access-token-read-role",
	}

	if len(options.tenantIDs) > 0 {
		for _, tID := range options.tenantIDs {
			roles = append(roles, tID+"_"+enReadWriteRole)
		}
	} else {
		roles = append(roles, enReadWriteRole)
	}
	claims := &jwt.MapClaims{
		"iss": "https://keycloak.kind.internal/realms/master",
		"exp": time.Now().Add(time.Hour).Unix(),
		"typ": "Bearer",
		"realm_access": map[string]interface{}{
			"roles": roles,
		},
	}

	return CreateJWTWithClaims(claims)
}

// CreateJWTWithClaims returns random signing key and JWT token (HS256 encoded) in a string with defined claims.
func CreateJWTWithClaims(claims *jwt.MapClaims) (string, string, error) {
	os.Setenv(SharedSecretKey, secretKey)
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims)
	jwtStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", "", err
	}
	return secretKey, jwtStr, nil
}
