// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agents_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/agents"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var zlog = logging.GetLogger("agents_test")

func Test_PuaSched_Single(t *testing.T) {
	zlog.Info().Msg("Test_PuaSched_Single Started")

	wg := sync.WaitGroup{}
	termChan := make(chan bool)
	stateChan := make(chan pb.UpdateStatus_StatusType, 1)
	respChan := make(chan *pb.PlatformUpdateStatusResponse)
	pua := agents.NewPUA(stateChan)
	pua.Handle(&wg, termChan, respChan)

	currentUpdateStatus := pua.State()
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE, currentUpdateStatus)

	respUpdate := &pb.PlatformUpdateStatusResponse{
		UpdateSchedule: &pb.UpdateSchedule{
			SingleSchedule: &utils_test.MmSingleSchedule1,
		},
		UpdateSource: &pb.UpdateSource{
			KernelCommand: utils_test.OSResource.GetKernelCommand(),
			OsRepoUrl:     utils_test.OSResource.GetImageUrl(),
			CustomRepos:   utils_test.OSResource.GetUpdateSources(),
		},
		InstalledPackages: utils_test.OSResource.GetInstalledPackages(),
	}
	respChan <- respUpdate

	time.Sleep(time.Second * 2)
	gotState := <-stateChan
	zlog.Info().Msgf("Updated PUA state %v", gotState)
	currentUpdateStatus = pua.State()
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_STARTED, currentUpdateStatus)

	time.Sleep(time.Second * 5)
	gotState = <-stateChan
	zlog.Info().Msgf("Updated PUA state %v", gotState)
	currentUpdateStatus = pua.State()
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, currentUpdateStatus)

	termChan <- true
	wg.Wait()
	zlog.Info().Msg("Test_PuaSched_Single Finished")
}

func Test_PuaSched_Repeated(t *testing.T) {
	zlog.Info().Msg("Test_PuaSched_Repeated Started")

	wg := sync.WaitGroup{}
	termChan := make(chan bool)
	stateChan := make(chan pb.UpdateStatus_StatusType, 1)
	respChan := make(chan *pb.PlatformUpdateStatusResponse)
	pua := agents.NewPUA(stateChan)
	pua.Handle(&wg, termChan, respChan)

	currentUpdateStatus := pua.State()
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE, currentUpdateStatus)

	respUpdate := &pb.PlatformUpdateStatusResponse{
		UpdateSchedule: &pb.UpdateSchedule{
			RepeatedSchedules: utils_test.MmRepeatedSchedule1,
		},
		UpdateSource: &pb.UpdateSource{
			KernelCommand: utils_test.OSResource.GetKernelCommand(),
			OsRepoUrl:     utils_test.OSResource.GetImageUrl(),
			CustomRepos:   utils_test.OSResource.GetUpdateSources(),
		},
		InstalledPackages: utils_test.OSResource.GetInstalledPackages(),
	}
	respChan <- respUpdate

	time.Sleep(time.Second * 2)
	gotState := <-stateChan
	zlog.Info().Msgf("Updated PUA state %v", gotState)
	currentUpdateStatus = pua.State()
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, currentUpdateStatus)

	termChan <- true
	wg.Wait()
	zlog.Info().Msg("Test_PuaSched_Repeated Finished")
}
