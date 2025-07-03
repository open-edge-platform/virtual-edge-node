// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/onboard/proto"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	enCredentialsPerm = 0o644
)

var (
	ctxTimeout     = time.Second * 10
	retryInterval  = time.Second * 1
	actionDuration = time.Millisecond * 10
)

var (
	errGetWfContext       = "failed to get workflow context"
	errGetWfActions       = "failed to get actions for workflow"
	errReportActionStatus = "failed to report action status: name %s, task name %s, status %s"
	errRcvWorkflowCtx     = "failed to rcv workflow contexts"
	msgTurn               = "it's turn for a different worker: %s"
)

const (
	actionClientID         = "write-client-id"
	actionClientSecret     = "write-client-secret"
	actionInstallCloudInit = "install-cloud-init"
)

type TinkWorker interface {
	ExecuteWorkflow(context.Context) error
}

type tinkWorker struct {
	workerID      string
	client        proto.WorkflowServiceClient
	retryInterval time.Duration
	settings      *defs.Settings
}

// UserData represents the cloud-config structure.
type UserData struct {
	Hostname           string      `yaml:"hostname,omitempty"`
	CreateHostnameFile bool        `yaml:"create_hostname_file,omitempty"`
	WriteFiles         []WriteFile `yaml:"write_files,omitempty"`
}

type WriteFile struct {
	Path        string `yaml:"path"`
	Owner       string `yaml:"owner,omitempty"`
	Permissions string `yaml:"permissions,omitempty"`
	Content     string `yaml:"content"`
}

func NewTinkWorker(workerID string, settings *defs.Settings) (TinkWorker, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
	defer cancel()

	kcToken, err := ensim_kc.GetKeycloakToken(
		ctx,
		settings.CertCA,
		settings.OrchFQDN,
		settings.EdgeOnboardUser,
		settings.EdgeOnboardPass,
		defs.OrchKcClientID,
	)
	if err != nil {
		zlog.Err(err).Msgf("failed to get keycloak API token")
		return nil, err
	}

	tinkerAddress := fmt.Sprintf("tinkerbell-server.%s:443", settings.OrchFQDN)
	conn, err := utils.Connect(tinkerAddress, settings.CertCAPath, kcToken)
	if err != nil {
		return nil, err
	}
	workflowClient := proto.NewWorkflowServiceClient(conn)

	w := &tinkWorker{
		workerID:      workerID,
		client:        workflowClient,
		retryInterval: retryInterval,
		settings:      settings,
	}
	return w, nil
}

// ExecuteWorkflow gets all Workflow contexts and processes their actions.
//
//nolint:cyclop,funlen // extracted from tink server repo.
func (t tinkWorker) ExecuteWorkflow(ctx context.Context) error {
	zlog.Info().Msgf("Execute Tinkerbell Workflow with workerID %s", t.workerID)
	workflowDone := make(chan bool)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-workflowDone:
			zlog.Info().Msgf("Tinkerbell workflow done with workerID %s", t.workerID)
			return nil
		default:
		}
		zlog.Debug().Msgf("GetWorkflowContexts %s", t.workerID)
		res, err := t.client.GetWorkflowContexts(ctx, &proto.WorkflowContextRequest{WorkerId: t.workerID})
		if err != nil {
			zlog.Error().Err(err).Msg(errGetWfContext)
			<-time.After(t.retryInterval)
			continue
		}
	workflowLoop:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-workflowDone:
				zlog.Info().Msgf("Tinkerbell actions done with workerID %s", t.workerID)
				break workflowLoop
			default:
			}
			wfContext, err := res.Recv()
			if err != nil || wfContext == nil {
				if !errors.Is(err, io.EOF) {
					zlog.Error().Err(err).Msg(errRcvWorkflowCtx)
					<-time.After(t.retryInterval)
				}
				break
			}
			wfID := wfContext.GetWorkflowId()
			actions, err := t.client.GetWorkflowActions(ctx, &proto.WorkflowActionsRequest{WorkflowId: wfID})
			if err != nil {
				zlog.Error().Err(err).Msg(errGetWfActions)
				continue
			}

			turn := false
			actionIndex := 0
			var nextAction *proto.WorkflowAction
			if wfContext.GetCurrentAction() == "" {
				if actions.GetActionList()[0].GetWorkerId() == t.workerID {
					actionIndex = 0
					turn = true
				}
			} else {
				switch wfContext.GetCurrentActionState() {
				case proto.State_STATE_SUCCESS:
					if isLastAction(wfContext, actions) {
						continue
					}
					nextAction = actions.GetActionList()[wfContext.GetCurrentActionIndex()+1]
					actionIndex = int(wfContext.GetCurrentActionIndex()) + 1
				case proto.State_STATE_FAILED:
					continue
				case proto.State_STATE_TIMEOUT:
					continue
				default:
					nextAction = actions.GetActionList()[wfContext.GetCurrentActionIndex()]
					actionIndex = int(wfContext.GetCurrentActionIndex())
				}
				if nextAction.GetWorkerId() == t.workerID {
					turn = true
				}
			}

			for turn {
				action := actions.GetActionList()[actionIndex]
				zlog.Debug().Msgf("Starting action: actionName %s taskName %s",
					action.GetName(), action.GetTaskName())

				if wfContext.GetCurrentActionState() != proto.State_STATE_RUNNING {
					actionStatus := &proto.WorkflowActionStatus{
						WorkflowId:   wfID,
						TaskName:     action.GetTaskName(),
						ActionName:   action.GetName(),
						ActionStatus: proto.State_STATE_RUNNING,
						Seconds:      0,
						Message:      "Started execution",
						WorkerId:     action.GetWorkerId(),
					}
					t.reportActionStatus(ctx, actionStatus)
					zlog.Debug().
						Msgf("Action: status %s duration %s",
							actionStatus.ActionStatus, strconv.FormatInt(actionStatus.Seconds, 10))
				}

				// start executing the action
				start := time.Now()
				st, err := t.execute(ctx, wfID, action)
				elapsed := time.Since(start)

				actionStatus := &proto.WorkflowActionStatus{
					WorkflowId: wfID,
					TaskName:   action.GetTaskName(),
					ActionName: action.GetName(),
					Seconds:    int64(elapsed.Seconds()),
					WorkerId:   action.GetWorkerId(),
				}

				if err != nil || st != proto.State_STATE_SUCCESS {
					if st == proto.State_STATE_TIMEOUT {
						actionStatus.ActionStatus = proto.State_STATE_TIMEOUT
					} else {
						actionStatus.ActionStatus = proto.State_STATE_FAILED
					}
					zlog.Error().Err(err).Msgf("failed to execute workflow action %s",
						action.GetName())
					t.reportActionStatus(ctx, actionStatus)
					break
				}

				actionStatus.ActionStatus = proto.State_STATE_SUCCESS
				actionStatus.Message = "finished execution successfully"
				t.reportActionStatus(ctx, actionStatus)

				if len(actions.GetActionList()) == actionIndex+1 {
					zlog.Debug().Msgf("reached to end of workflow")
					close(workflowDone)
					break
				}

				nextAction := actions.GetActionList()[actionIndex+1]
				if nextAction.GetWorkerId() != t.workerID {
					zlog.Debug().Msgf(msgTurn, nextAction.GetWorkerId())
					turn = false
				} else {
					actionIndex++
				}
			}
		}
		// wait for the next try to get workflow
		select {
		case <-ctx.Done():
			return nil
		case <-workflowDone:
			zlog.Info().Msgf("Tinkerbell workflow done with workerID %s", t.workerID)
			return nil
		case <-time.After(t.retryInterval):
			zlog.Debug().Msgf("waiting for workflow %s", t.workerID)
		}
	}
}

func isLastAction(wfContext *proto.WorkflowContext, actions *proto.WorkflowActionList) bool {
	return int(wfContext.GetCurrentActionIndex()) == len(actions.GetActionList())-1
}

// reportActionStatus reports the status of an action to the Tinkerbell server and retries forever on error.
func (t *tinkWorker) reportActionStatus(ctx context.Context, actionStatus *proto.WorkflowActionStatus) {
	for {
		zlog.Debug().
			Msgf("reporting Action Status: name %s, task name %s, status %s",
				actionStatus.GetActionName(), actionStatus.GetTaskName(), actionStatus.GetActionStatus())
		_, err := t.client.ReportActionStatus(ctx, actionStatus)
		if err != nil {
			zlog.Error().
				Err(err).
				Msgf(errReportActionStatus,
					actionStatus.GetActionName(),
					actionStatus.GetTaskName(),
					actionStatus.GetActionStatus())
			<-time.After(t.retryInterval)
			continue
		}
		return
	}
}

func (t *tinkWorker) execute(_ context.Context, wfID string, action *proto.WorkflowAction) (proto.State, error) {
	time.Sleep(actionDuration)
	var err error
	st := proto.State_STATE_SUCCESS
	zlog.Debug().Msgf("Action: name %s command %s: env: %s",
		action.GetName(), action.GetCommand(), action.GetEnvironment())
	switch action.GetName() {
	case actionInstallCloudInit:
		st, err = t.executeCloudInit(action)
	case actionClientID:
		st, err = t.executeActionWriteFile(action)
	case actionClientSecret:
		st, err = t.executeActionWriteFile(action)
	}
	zlog.Debug().
		Msgf("Action DONE: executed workflowID %s, name %s, task name %s, status %s",
			wfID, action.GetName(), action.GetTaskName(), st.String())
	return st, err
}

func (t *tinkWorker) getActionEnvContents(action *proto.WorkflowAction) (string, error) {
	var envContents string
	actEnv := action.GetEnvironment()
	for _, env := range actEnv {
		if strings.Contains(env, "CONTENTS") {
			envContents = strings.TrimPrefix(env, "CONTENTS=")
			break
		}
	}

	if envContents == "" {
		return "", fmt.Errorf("could not find CONTENTS in environment %v", actEnv)
	}
	return envContents, nil
}

// executeActionWriteFile writes the client ID and secret to the respective files.
// It can be extended to write other files given the action name.
func (t *tinkWorker) executeActionWriteFile(action *proto.WorkflowAction) (proto.State, error) {
	contents, err := t.getActionEnvContents(action)
	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	if action.GetName() == actionClientID {
		err = os.WriteFile(t.settings.BaseFolder+defs.ENClientIDPath, []byte(contents+"\n"), enCredentialsPerm)
		if err != nil {
			zlog.Err(err).Msgf("could not write client ID file %s", t.settings.BaseFolder+defs.ENClientIDPath)
		}
	}
	if action.GetName() == actionClientSecret {
		err = os.WriteFile(t.settings.BaseFolder+defs.ENClientSecretPath, []byte(contents+"\n"), enCredentialsPerm)
		if err != nil {
			zlog.Err(err).
				Msgf("could not write client secret file %s", t.settings.BaseFolder+defs.ENClientSecretPath)
		}
	}

	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	return proto.State_STATE_SUCCESS, nil
}

// executeCloudInitUserDataHostname sets the hostname of the system based on the provided user data.
// It only sets the hostname if the ENiC (Edge Node in Cloud) feature is enabled.
//
//nolint:gosec // This function is not vulnerable to command injection as it uses a trusted source for the hostname.
func (t *tinkWorker) executeCloudInitUserDataHostname(userData UserData) error {
	// Only sets hostname if ENiC is enabled
	if !t.settings.ENiC {
		return nil
	}

	if userData.CreateHostnameFile && userData.Hostname != "" {
		zlog.Debug().Msgf("Set hostname: %s", userData.Hostname)

		// Update the hostname using the `hostnamectl` command
		cmd := exec.Command("hostname", userData.Hostname)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set hostname using hostname cmd: %w", err)
		}

		// Update the /etc/hostname file
		err := os.WriteFile("/etc/hostname", []byte(userData.Hostname+"\n"), enCredentialsPerm)
		if err != nil {
			return fmt.Errorf("failed to write to /etc/hostname: %w", err)
		}
	}
	return nil
}

// executeCloudInitUserData writes the client ID, secret, and tenantId to the respective files.
//
//nolint:cyclop // extracted from cloud-init repo.
func (t *tinkWorker) executeCloudInitUserData(userData UserData) error {
	var err error
	foundClientID := false
	foundClientSecret := false
	for _, writeFile := range userData.WriteFiles {
		if strings.Contains(writeFile.Path, defs.ENClientIDPath) {
			foundClientID = true
			err = os.WriteFile(t.settings.BaseFolder+defs.ENClientIDPath, []byte(writeFile.Content+"\n"), enCredentialsPerm)
			if err != nil {
				zlog.Err(err).Msgf("could not write client ID file %s", t.settings.BaseFolder+defs.ENClientIDPath)
			}
		}
		if strings.Contains(writeFile.Path, defs.ENClientSecretPath) {
			foundClientSecret = true
			err = os.WriteFile(t.settings.BaseFolder+defs.ENClientSecretPath, []byte(writeFile.Content+"\n"), enCredentialsPerm)
			if err != nil {
				zlog.Err(err).
					Msgf("could not write client secret file %s", t.settings.BaseFolder+defs.ENClientSecretPath)
			}
		}
		if strings.Contains(writeFile.Path, defs.ENTenantIDPath) {
			err = os.WriteFile(t.settings.BaseFolder+defs.ENTenantIDPath, []byte(writeFile.Content+"\n"), enCredentialsPerm)
			if err != nil {
				zlog.Err(err).
					Msgf("could not write client tenantID file %s", t.settings.BaseFolder+defs.ENTenantIDPath)
			}
		}
	}

	if err != nil {
		return err
	}

	if !foundClientID || !foundClientSecret {
		return fmt.Errorf("client ID or secret not found in cloud-init")
	}

	return nil
}

func (t *tinkWorker) executeCloudInit(action *proto.WorkflowAction) (proto.State, error) {
	contents, err := t.getActionEnvContents(action)
	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	// remove #cloud-config for successful unmarshalling
	lines := strings.Split(contents, "\n")
	var result []string
	for _, line := range lines {
		if !strings.Contains(line, "#cloud-config") {
			result = append(result, line)
		}
	}
	contents = strings.Join(result, "\n")

	var userData UserData
	err = yaml.Unmarshal([]byte(contents), &userData)
	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	err = t.executeCloudInitUserData(userData)
	if err != nil {
		zlog.Err(err).Msgf("failed to execute cloud-init user data")
		return proto.State_STATE_FAILED, err
	}

	err = t.executeCloudInitUserDataHostname(userData)
	if err != nil {
		zlog.Err(err).Msgf("failed to set hostname")
		return proto.State_STATE_FAILED, err
	}

	return proto.State_STATE_SUCCESS, nil
}
