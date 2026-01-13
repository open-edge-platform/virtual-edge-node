// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"google.golang.org/protobuf/proto"

	inv_util "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
)

var kernelRegexp = regexp.MustCompile(`(^[A-Za-z0-9-_=.,/ ]*$)`)

const (
	repeatedScheduleTag = "RepeatedSchedule"
	singleScheduleTag   = "SingleSchedule"
	updateDuration      = 30 * time.Second
)

type Pua struct {
	mu                     sync.RWMutex
	scheduler              *gocron.Scheduler
	state                  pb.UpdateStatus_StatusType
	stateChan              chan pb.UpdateStatus_StatusType
	singleSchedule         *pb.SingleSchedule
	singleScheduleFinished bool
	repeatedSchedules      []*pb.RepeatedSchedule
}

func NewPUA(stateChan chan pb.UpdateStatus_StatusType) *Pua {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.SingletonModeAll()
	scheduler.StartAsync()
	scheduler.SetMaxConcurrentJobs(1, gocron.WaitMode)

	return &Pua{
		scheduler:              scheduler,
		state:                  pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE,
		stateChan:              stateChan,
		singleSchedule:         nil,
		singleScheduleFinished: false,
		repeatedSchedules:      []*pb.RepeatedSchedule{},
	}
}

func (p *Pua) State() pb.UpdateStatus_StatusType {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Pua) updateState(s pb.UpdateStatus_StatusType) {
	p.mu.Lock()
	p.state = s
	p.mu.Unlock()
	p.stateChan <- s
}

func (p *Pua) UpToDate() {
	p.mu.Lock()
	p.state = pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE
	p.mu.Unlock()
}

func (p *Pua) Handle(wg *sync.WaitGroup, termChan chan bool, updateResChan chan *pb.PlatformUpdateStatusResponse) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-termChan:
				zlog.Info().Msgf("Update Agent - Terminating pua Handle routine")
				return

			case updateRes := <-updateResChan:
				if updateRes.UpdateSchedule != nil {
					p.handleUpdateRes(updateRes)
				} else {
					zlog.Warn().Msgf("got nil UpdateSchedule - skipping scheduling")
					p.CleanupSchedule()
				}
			}
		}
	}()
}

func (p *Pua) handleUpdateRes(updateRes *pb.PlatformUpdateStatusResponse) {
	switch {
	case updateRes.GetUpdateSource() == nil || updateRes.GetUpdateSchedule() == nil:
		zlog.Warn().Msgf("skipping response as it is missing one of the fields")
		zlog.Debug().Msgf("skipped response - %v", updateRes)
	case kernelRegexp.MatchString(updateRes.UpdateSource.KernelCommand):
		p.handleSingleSchedule(updateRes.UpdateSchedule.SingleSchedule)
		p.handleRepeatedSchedule(updateRes.UpdateSchedule.RepeatedSchedules)
	case !kernelRegexp.MatchString(updateRes.UpdateSource.KernelCommand):
		zlog.Error().
			Msgf("skipping request, as '%v' kernel command violates safe kernel settings", updateRes.UpdateSource.KernelCommand)
	default:
		zlog.Debug().Msgf("unexpected case of default in switch - %v", updateRes)
	}
}

func (p *Pua) handleSingleSchedule(schedule *pb.SingleSchedule) {
	switch {
	case schedule == nil:
		zlog.Debug().Msgf("'%v' has no schedule/job", singleScheduleTag)
		err := p.scheduler.RemoveByTag(singleScheduleTag)
		if err != nil {
			zlog.Debug().Msgf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
		}
		p.singleSchedule = nil
		p.singleScheduleFinished = true

	case !(proto.Equal(p.singleSchedule, schedule) && p.singleScheduleFinished): //nolint:staticcheck // left for clarity
		zlog.Debug().Msgf("'%v' has schedule/job %v", singleScheduleTag, schedule)
		jobs, errJobs := p.scheduler.FindJobsByTag(
			singleScheduleTag,
		) // error is ignored intentionally as it only returns ErrJobNotFoundWithTag
		if errJobs != nil {
			zlog.Debug().Msgf("scheduler failed to find cron jobs by '%v' tag - %v", singleScheduleTag, errJobs)
		}
		switch {
		case len(jobs) == 0:
			p.scheduleSingleRun(schedule)
			p.singleSchedule = schedule
			p.singleScheduleFinished = false
		default:
			if jobs[0].FinishedRunCount() == 1 {
				p.singleScheduleFinished = true
				zlog.Info().Msg("marking single schedule job as done")
			}
		}
	}
}

func (p *Pua) removeRepeatedSchedules() {
	err := p.scheduler.RemoveByTag(repeatedScheduleTag)
	if err != nil {
		zlog.Debug().Msgf("scheduler failed to remove cron jobs by '%v' tag - %v", repeatedScheduleTag, err)
	}
}

// scheduleRepeatedSchedule schedules a repeated schedule job to be executed according
// to the cron definitions of it.
// The job mimics the maintenance of an OS, with a fixed duration.
func (p *Pua) scheduleRepeatedSchedule(schedule *pb.RepeatedSchedule) {
	_, err := p.scheduler.Tag(repeatedScheduleTag).Cron(CronScheduleToString(schedule)).
		Do(func() {
			zlog.Debug().Msgf("update is triggered by %v", repeatedScheduleTag)
			p.updateState(pb.UpdateStatus_STATUS_TYPE_STARTED)
			// Random sleep to simulate update task
			time.Sleep(updateDuration)
			p.updateState(pb.UpdateStatus_STATUS_TYPE_UPDATED)
		})
	if err != nil {
		zlog.Error().Msgf("failed to schedule cron job - %v", err)
		p.updateState(pb.UpdateStatus_STATUS_TYPE_FAILED)
	}
}

func (p *Pua) scheduleSingleRun(schedule *pb.SingleSchedule) {
	err := p.scheduler.RemoveByTag(singleScheduleTag)
	if err != nil {
		zlog.Debug().Msgf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
	}
	startSecs, err := inv_util.Uint64ToInt64(schedule.StartSeconds)
	if err != nil {
		zlog.Error().Msgf("failed to convert start seconds to int64 - %v", err)
		p.updateState(pb.UpdateStatus_STATUS_TYPE_FAILED)
		return
	}
	startTime := time.Unix(startSecs, 0)
	durationSeconds := uint64(0)
	if schedule.EndSeconds != 0 {
		durationSeconds = schedule.EndSeconds - schedule.StartSeconds
	}

	// If a single schedule is defined, it will be done by gcronx with a start time defined as in the schedule
	// to be executed just once.
	// The schedule mimics the execution of an OS update (e.g., installing packages).
	zlog.Debug().Msgf("Scheduler - update is scheduled by %v for %v at %v", singleScheduleTag, durationSeconds, startTime)
	_, err = p.scheduler.Every(1).Day().Tag(singleScheduleTag).StartAt(startTime).LimitRunsTo(1).Do(func() {
		zlog.Debug().Msgf("update is triggered by %v for %v", singleScheduleTag, durationSeconds)
		p.updateState(pb.UpdateStatus_STATUS_TYPE_STARTED)
		// Random sleep to simulate update task
		time.Sleep(updateDuration)
		p.updateState(pb.UpdateStatus_STATUS_TYPE_UPDATED)
	})
	if err != nil {
		zlog.Error().Msgf("failed to schedule single schedule job - %v", err)
		p.updateState(pb.UpdateStatus_STATUS_TYPE_FAILED)
	}
}

//nolint:cyclop // this function will be refactored
func (p *Pua) handleRepeatedSchedule(schedules []*pb.RepeatedSchedule) {
	jobs, err := p.scheduler.FindJobsByTag(repeatedScheduleTag)
	if err != nil {
		zlog.Debug().Msgf("scheduler failed to find cron jobs by '%v' tag - %v", repeatedScheduleTag, err)
	}
	if len(schedules) == 0 {
		zlog.Debug().Msgf("'%v' has no schedule/job", repeatedScheduleTag)
		p.repeatedSchedules = nil
		err := p.scheduler.RemoveByTag(repeatedScheduleTag)
		if err != nil {
			zlog.Debug().Msgf("scheduler failed to remove cron job by '%v' tag - %v", repeatedScheduleTag, err)
		}
	}
	exists := false

	if len(jobs) != 0 && (len(schedules) == len(p.repeatedSchedules)) {
		for _, schedule := range schedules {
			exists = false
			for _, metaSchedule := range p.repeatedSchedules {
				if proto.Equal(schedule, metaSchedule) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			break
		}
	}

	if !exists {
		if len(jobs) != 0 {
			p.removeRepeatedSchedules()
		}
		zlog.Debug().Msgf("'%v' has schedule/job %v", repeatedScheduleTag, schedules)
		p.repeatedSchedules = []*pb.RepeatedSchedule{}
		for _, schedule := range schedules {
			p.scheduleRepeatedSchedule(schedule)
			p.repeatedSchedules = append(p.repeatedSchedules, schedule)
		}
	}
}

func (p *Pua) CleanupSchedule() {
	err := p.scheduler.RemoveByTag(singleScheduleTag)
	if err != nil {
		zlog.Debug().Msgf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
	}
	err = p.scheduler.RemoveByTag(repeatedScheduleTag)
	if err != nil {
		zlog.Debug().Msgf("scheduler failed to remove cron job by '%v' tag - %v", repeatedScheduleTag, err)
	}
}

func (p *Pua) GetJobs() []*gocron.Job {
	return p.scheduler.Jobs()
}

func CronScheduleToString(schedule *pb.RepeatedSchedule) string {
	return fmt.Sprintf("%v %v %v %v %v", schedule.GetCronMinutes(), schedule.GetCronHours(),
		schedule.GetCronDayMonth(), schedule.GetCronMonth(), schedule.GetCronDayWeek())
}
