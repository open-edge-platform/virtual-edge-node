// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package mt_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var zlog = logging.GetLogger("mt")

var (
	// clusterFQDN  = "kind.internal".
	clusterFQDN  = "kind.internal"
	keycloakURL  = "https://keycloak." + clusterFQDN
	keycloakUser = ""
	keycloakPass = "" // update <default-password>
	realm        = "master"
	caPath       = "ca.crt"

	orgName      = "testOrg1"
	projectName  = "testProject1"
	userName     = "testUser1"
	userPassword = "testPassword1!"

	adminUserName     = "adminTestUser1"
	adminUserPassword = "adminTestPassword1!"

	customerID = "150238189"
	productKey = "0eb02da9-daa6-bbce-efb1-62800ff42489"
)

type Org struct {
	Spec   map[string]interface{} `json:"spec"`
	Status map[string]interface{} `json:"status"`
}

type Project struct {
	Name   string                 `json:"name"`
	Spec   map[string]interface{} `json:"spec"`
	Status map[string]interface{} `json:"status"`
}

// TestMT_Case01 Create usable org, project, and users for Infrastructure Manager simulator.
func TestMT_Case01(t *testing.T) {
	zlog.Info().Msg("TestMT_Case01 Started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 0. Set up certCA
	certCA, err := utils_test.LoadFile(caPath)
	require.NoError(t, err)
	require.NotNil(t, certCA)

	// 1. Login and get admin token
	kcAdmin, err := utils_test.NewKeycloakService(ctx, keycloakURL, keycloakUser, keycloakPass, realm, caPath)
	require.NoError(t, err)
	require.NotNil(t, kcAdmin)
	defer func() {
		errLogout := kcAdmin.Logout(ctx, userName, realm)
		assert.NoError(t, errLogout)
	}()

	// 2. Get token
	tokenAdmin, err := kcAdmin.GetToken(ctx, realm, keycloakUser, keycloakPass)
	require.NoError(t, err)
	zlog.Info().Msgf("TestMT_Case01 tokenAdmin %s", tokenAdmin)

	// 3. Create org
	orgStat, err := utils_test.CreateOrg(ctx, certCA, clusterFQDN, tokenAdmin, orgName)
	require.NoError(t, err)
	require.NotNil(t, orgStat)

	time.Sleep(30 * time.Second)

	orgStatGet, err := utils_test.GetOrg(ctx, certCA, clusterFQDN, tokenAdmin, orgName)
	require.NoError(t, err)
	require.NotNil(t, orgStatGet)

	// 4. Get org uuid
	orgStruct := Org{}
	err = json.Unmarshal([]byte(orgStatGet), &orgStruct)
	require.NoError(t, err)

	orgStatus, ok := orgStruct.Status["orgStatus"].(map[string]interface{})
	require.True(t, ok)
	require.NotNil(t, orgStatus)

	orgUUID, ok := orgStatus["uID"].(string)
	require.True(t, ok)
	require.NotNil(t, orgUUID)
	orgStatusStr, ok := orgStatus["statusIndicator"].(string)
	require.True(t, ok)
	require.Equal(t, "STATUS_INDICATION_IDLE", orgStatusStr)

	// 5. Create License for the Org
	err = utils_test.CreateOrgLicense(ctx, certCA, clusterFQDN, tokenAdmin, orgName, customerID, productKey)
	require.NoError(t, err)

	// 6. Create admin user for the org
	userID, err := kcAdmin.CreateUser(ctx, realm, userName, userPassword)
	require.NoError(t, err)
	require.NotNil(t, userID)

	// 7. Add user to org
	groupName := orgUUID + "_Project-Manager-Group"
	err = kcAdmin.AddUserToGroup(ctx, realm, userName, groupName)
	require.NoError(t, err)

	// 8. Login and get user token
	kcUser, err := utils_test.NewKeycloakService(ctx, keycloakURL, userName, userPassword, realm, caPath)
	require.NoError(t, err)
	require.NotNil(t, kcUser)
	defer func() {
		errLogout := kcUser.Logout(ctx, userName, realm)
		assert.NoError(t, errLogout)
	}()

	tokenUser, err := kcAdmin.GetToken(ctx, realm, userName, userPassword)
	require.NoError(t, err)
	require.NotNil(t, tokenUser)

	// 9. Create project for the Org
	projectStatus, err := utils_test.CreateProject(ctx, certCA, clusterFQDN, tokenUser, projectName)
	require.NoError(t, err)
	require.NotNil(t, projectStatus)

	time.Sleep(30 * time.Second)

	// 10. Make sure project is created and get proj uuid
	projectStat, err := utils_test.GetProject(ctx, certCA, clusterFQDN, tokenUser, projectName)
	require.NoError(t, err)

	projStruct := Project{}
	err = json.Unmarshal([]byte(projectStat), &projStruct)
	require.NoError(t, err)

	projStatus, ok := projStruct.Status["projectStatus"].(map[string]interface{})
	require.True(t, ok)

	projUUID, ok := projStatus["uID"].(string)
	require.True(t, ok)
	require.NotNil(t, projUUID)
	projStatStr, ok := projStatus["statusIndicator"].(string)
	require.True(t, ok)
	require.Equal(t, "STATUS_INDICATION_IDLE", projStatStr)

	// 11. Create admin user for the project
	adminUserID, err := kcAdmin.CreateUser(ctx, realm, adminUserName, adminUserPassword)
	require.NoError(t, err)
	require.NotNil(t, adminUserID)

	// 12. Add proj admin user to groups
	groupNames := []string{
		projUUID + "_Edge-Manager-Group",
		projUUID + "_Edge-Onboarding-Group",
		projUUID + "_Edge-Operator-Group",
		projUUID + "_Host-Manager-Group",
		orgUUID + "_Project-Manager-Group",
	}
	for _, groupName := range groupNames {
		err = kcAdmin.AddUserToGroup(ctx, realm, adminUserName, groupName)
		require.NoError(t, err)
	}
}

// TestMT_Case02 Delete org, project, and users created in TestMT_Case01.
func TestMT_Case02(t *testing.T) {
	zlog.Info().Msg("TestMT_Case01 Started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up certCA
	certCA, err := utils_test.LoadFile(caPath)
	require.NoError(t, err)
	require.NotNil(t, certCA)

	// Login and get admin token
	kcAdmin, err := utils_test.NewKeycloakService(ctx, keycloakURL, keycloakUser, keycloakPass, realm, caPath)
	require.NoError(t, err)
	require.NotNil(t, kcAdmin)
	defer func() {
		errLogout := kcAdmin.Logout(ctx, userName, realm)
		assert.NoError(t, errLogout)
	}()

	// Get token
	tokenAdmin, err := kcAdmin.GetToken(ctx, realm, keycloakUser, keycloakPass)
	require.NoError(t, err)
	zlog.Info().Msgf("TestMT_Case01 tokenAdmin %s", tokenAdmin)

	// List Orgs
	orgs, err := utils_test.ListOrgs(ctx, certCA, clusterFQDN, tokenAdmin)
	require.NoError(t, err)
	require.NotNil(t, orgs)
	zlog.Info().Msgf("TestMT_Case01 orgs %s", orgs)

	// List Projects
	projs, err := utils_test.ListProjects(ctx, certCA, clusterFQDN, tokenAdmin)
	require.NoError(t, err)
	require.NotNil(t, projs)
	zlog.Info().Msgf("TestMT_Case01 projs %s", projs)

	// Login and get user token
	kcUser, err := utils_test.NewKeycloakService(ctx, keycloakURL, userName, userPassword, realm, caPath)
	require.NoError(t, err)
	require.NotNil(t, kcUser)
	defer func() {
		errLogout := kcUser.Logout(ctx, userName, realm)
		assert.NoError(t, errLogout)
	}()

	tokenUser, err := kcAdmin.GetToken(ctx, realm, userName, userPassword)
	require.NoError(t, err)
	require.NotNil(t, tokenUser)

	// Delete project
	err = utils_test.DeleteProject(ctx, certCA, clusterFQDN, tokenUser, projectName)
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// Delete org license
	orgLicenses, err := utils_test.GetOrgLicenses(ctx, certCA, clusterFQDN, tokenAdmin, orgName)
	require.NoError(t, err)
	require.NotNil(t, orgLicenses)

	err = utils_test.DeleteOrgLicense(ctx, certCA, clusterFQDN, tokenAdmin, orgName)
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// Delete org
	err = utils_test.DeleteOrg(ctx, certCA, clusterFQDN, tokenAdmin, orgName)
	require.NoError(t, err)

	// Delete project admin user
	err = kcAdmin.DeleteUser(ctx, realm, adminUserName)
	require.NoError(t, err)

	// Delete org admin user
	err = kcAdmin.DeleteUser(ctx, realm, userName)
	require.NoError(t, err)
}
