// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	hostmgr "github.com/open-edge-platform/infra-managers/host/pkg/api/hostmgr/proto"
	maintmgr "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	telmgr "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

var zlog = logging.GetLogger("agents")

const (
	backoffRetries  = 3
	backoffInterval = time.Second * 10
)

var (
	nodeAgentInterval      = time.Second * 30
	updateAgentInterval    = time.Second * 300
	hdaInterval            = time.Second * 60
	telemetryAgentInterval = time.Second * 60
	connRenewInterval      = time.Minute * 55
)

type AgentState int

const (
	AgentStateUnknown AgentState = iota
	AgentStateOn                 // Agent turned on
	AgentStateOff                // Agent turned off
)

type AgentType int

const (
	AgentTypeUnknown AgentType = iota
	AgentTypeNode
	AgentTypeUpdate
	AgentTypeHD
	AgentTypeTelemetry
)

var agentTypeToString = map[AgentType]string{
	AgentTypeNode:      "Node",
	AgentTypeUpdate:    "Update",
	AgentTypeHD:        "HD",
	AgentTypeTelemetry: "Telemetry",
}

func (at AgentType) String() string {
	if str, ok := agentTypeToString[at]; ok {
		return str
	}
	return "Unknown"
}

type StateMap struct {
	Current map[AgentType]AgentState // Defines the current state of the agents
	Desired map[AgentType]AgentState // Defines the desired state of the agents
}
type State struct {
	Current sync.Map // Defines the current state of the agents - map[AgentType]AgentState
	Desired sync.Map // Defines the desired state of the agents - map[AgentType]AgentState
}

var (
	agentsStatesCurrentInit = map[AgentType]AgentState{
		AgentTypeNode:      AgentStateOn,
		AgentTypeUpdate:    AgentStateOn,
		AgentTypeHD:        AgentStateOn,
		AgentTypeTelemetry: AgentStateOn,
	}
	agentsStatesDesiredInit = map[AgentType]AgentState{
		AgentTypeNode:      AgentStateOn,
		AgentTypeUpdate:    AgentStateOn,
		AgentTypeHD:        AgentStateOn,
		AgentTypeTelemetry: AgentStateOn,
	}
)

type Agents struct {
	wg              *sync.WaitGroup
	readyChan       chan bool
	termChan        chan bool
	statsChan       chan *ensimapi.NodeStatus
	cfg             *defs.Settings
	settings        *State
	hwInfo          *hostmgr.UpdateHostSystemInfoByGUIDRequest
	agentsConn      sync.Map
	agentsClient    sync.Map
	agentsAddresses map[AgentType]string
	pua             *Pua
	puaRespChan     chan *maintmgr.PlatformUpdateStatusResponse
	puaStateChan    chan maintmgr.UpdateStatus_StatusType
}

func (a *Agents) SetDesiredStates(states map[AgentType]AgentState) {
	changed := false
	for k, v := range states {
		// Checks if the agent exists and if the desired state is different from the set state
		cv, ok := a.settings.Desired.Load(k)
		if ok {
			agSt, ok := cv.(AgentState)
			if ok && agSt != v {
				a.settings.Desired.Store(k, v)
				changed = true
			}
		}
	}
	if changed {
		a.reconcile()
	}
}

func (a *Agents) GetStates() StateMap {
	m := StateMap{
		Current: make(map[AgentType]AgentState),
		Desired: make(map[AgentType]AgentState),
	}
	a.settings.Current.Range(func(k, v interface{}) bool {
		s, ok := v.(AgentState)
		if !ok {
			zlog.Error().Msgf("Agent %s - failed to cast state: UUID %s", k, a.cfg.ENGUID)
			return false
		}
		t, ok := k.(AgentType)
		if !ok {
			zlog.Error().Msgf("Agent %s - failed to cast type: UUID %s", k, a.cfg.ENGUID)
			return false
		}
		m.Current[t] = s
		return true
	})
	a.settings.Desired.Range(func(k, v interface{}) bool {
		s, ok := v.(AgentState)
		if !ok {
			zlog.Error().Msgf("Agent %s - failed to cast state: UUID %s", k, a.cfg.ENGUID)
			return false
		}
		t, ok := k.(AgentType)
		if !ok {
			zlog.Error().Msgf("Agent %s - failed to cast type: UUID %s", k, a.cfg.ENGUID)
			return false
		}
		m.Desired[t] = s
		return true
	})
	return m
}

func (a *Agents) GetCurrentState(agent AgentType) (AgentState, error) {
	if state, ok := a.settings.Current.Load(agent); ok {
		st, ok := state.(AgentState)
		if !ok {
			return AgentStateUnknown, fmt.Errorf("failed to cast state for agent %s: UUID %s", agent, a.cfg.ENGUID)
		}
		return st, nil
	}
	return AgentStateUnknown, fmt.Errorf("agent %s not found", agent)
}

func (a *Agents) skipAgentRoutine(agentType AgentType) bool {
	agentCurrentState, err := a.GetCurrentState(agentType)
	if err != nil {
		zlog.Error().Err(err).Msgf("Agent %s - failed to get current state: UUID %s", agentType, a.cfg.ENGUID)
		return false
	}
	if agentCurrentState == AgentStateOff {
		zlog.Info().Msgf("UUID %s - agent %s skiping routine, current state: %v", a.cfg.ENGUID, agentType, agentCurrentState)
		return true
	}
	return false
}

func (a *Agents) reconcile() {
	currentState := make(map[AgentType]AgentState)
	a.settings.Desired.Range(func(k, v interface{}) bool {
		if desExistingState, desExists := a.settings.Current.Load(k); desExists {
			existState, ok := desExistingState.(AgentState)
			if !ok {
				zlog.Error().Msgf("Agent %s - failed to cast state: UUID %s", k, a.cfg.ENGUID)
				return false
			}

			s, ok := v.(AgentState)
			if !ok {
				zlog.Error().Msgf("Agent %s - failed to cast state: UUID %s", k, a.cfg.ENGUID)
				return false
			}

			if existState != s {
				a.settings.Current.Swap(k, v)
				t, ok := k.(AgentType)
				if !ok {
					zlog.Error().Msgf("Agent %s - failed to cast agent type: UUID %s", k, a.cfg.ENGUID)
					return false
				}
				currentState[t] = s
			}
		}
		return true
	})
	zlog.Info().Msgf("Agents current state reconciled for UUID %s: %v", a.cfg.ENGUID, currentState)
}

func NewAgents(
	wg *sync.WaitGroup,
	readyChan, termChan chan bool,
	statsChan chan *ensimapi.NodeStatus,
	cfg *defs.Settings,
) *Agents {
	hwInfo := getHostSystemInfoMessage(cfg.ENGUID, cfg.ENSerial)
	settings := &State{
		Current: sync.Map{},
		Desired: sync.Map{},
	}

	for k, v := range agentsStatesCurrentInit {
		settings.Current.Store(k, v)
	}

	for k, v := range agentsStatesDesiredInit {
		settings.Desired.Store(k, v)
	}

	agentsAddresses := map[AgentType]string{
		AgentTypeNode:      fmt.Sprintf("infra-node.%s:443", cfg.OrchFQDN),
		AgentTypeHD:        fmt.Sprintf("infra-node.%s:443", cfg.OrchFQDN),
		AgentTypeUpdate:    fmt.Sprintf("update-node.%s:443", cfg.OrchFQDN),
		AgentTypeTelemetry: fmt.Sprintf("telemetry-node.%s:443", cfg.OrchFQDN),
	}

	puaRespChan := make(chan *maintmgr.PlatformUpdateStatusResponse)
	puaStateChan := make(chan maintmgr.UpdateStatus_StatusType)
	pua := NewPUA(puaStateChan)

	agnts := &Agents{
		wg:              wg,
		readyChan:       readyChan,
		termChan:        termChan,
		statsChan:       statsChan,
		cfg:             cfg,
		settings:        settings,
		hwInfo:          hwInfo,
		agentsConn:      sync.Map{},
		agentsClient:    sync.Map{},
		agentsAddresses: agentsAddresses,
		pua:             pua,
		puaRespChan:     puaRespChan,
		puaStateChan:    puaStateChan,
	}

	return agnts
}

func (a *Agents) initClients() error {
	for agentType := range a.agentsAddresses {
		if err := backoff.Retry(func() error {
			return a.instantiateClient(agentType, false)
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffInterval), backoffRetries)); err != nil {
			zlog.Error().Err(err).Msgf("failed to connect client %s: UUID %s", agentType, a.cfg.ENGUID)
			return err
		}
	}
	return nil
}

func (a *Agents) connectClient(agentType AgentType, reconnect bool) (*grpc.ClientConn, error) {
	zlog.Debug().Msgf("Agent %s - connect client to InfrastructureManager: UUID %s", agentType, a.cfg.ENGUID)
	if reconnect {
		connValue, hasConn := a.agentsConn.Load(agentType)
		if !hasConn {
			err := fmt.Errorf("connection for agent type %s not found: UUID %s", agentType, a.cfg.ENGUID)
			zlog.Warn().Err(err).Msgf("failed to reconnect client %s: UUID %s", agentType, a.cfg.ENGUID)
			return nil, err
		}
		conn, isConn := connValue.(*grpc.ClientConn)
		if !isConn {
			err := fmt.Errorf("could not parse connection for agent type %s not found: UUID %s", agentType, a.cfg.ENGUID)
			zlog.Warn().Err(err).Msgf("failed to reconnect client %s: UUID %s", agentType, a.cfg.ENGUID)
			return nil, err
		}
		connErr := conn.Close()
		if connErr != nil {
			zlog.Warn().Err(connErr).Msgf("failed to close connection for client %s: UUID %s", agentType, a.cfg.ENGUID)
		}
	}
	agentAddress, exists := a.agentsAddresses[agentType]
	if !exists {
		err := fmt.Errorf("agent type %s not found: UUID %s", agentType, a.cfg.ENGUID)
		zlog.Error().Err(err).Msgf("failed to connect client %s: UUID %s", agentType, a.cfg.ENGUID)
		return nil, err
	}
	conn, err := connect(agentAddress, a.cfg.CertCAPath)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to connect client %s: UUID %s", agentType, a.cfg.ENGUID)
		return nil, err
	}
	zlog.Debug().Msgf("Agent %s - successfully connected client to InfrastructureManager: UUID %s", agentType, a.cfg.ENGUID)
	a.agentsConn.Store(agentType, conn)
	return conn, nil
}

func (a *Agents) instantiateClient(agentType AgentType, reconnect bool) error {
	zlog.Debug().Msgf("Agent %s - Instantiating client to InfrastructureManager: UUID %s", agentType, a.cfg.ENGUID)
	conn, err := a.connectClient(agentType, reconnect)
	if err != nil {
		return err
	}

	var client interface{}
	switch agentType {
	case AgentTypeNode:
		client = hostmgr.NewHostmgrClient(conn)
	case AgentTypeHD:
		client = hostmgr.NewHostmgrClient(conn)
	case AgentTypeUpdate:
		client = maintmgr.NewMaintmgrServiceClient(conn)
	case AgentTypeTelemetry:
		client = telmgr.NewTelemetryMgrClient(conn)
	default:
		err := fmt.Errorf("agent type %s not found", agentType)
		zlog.Error().Err(err).Msgf("failed to instantiate client %s: UUID %s", agentType, a.cfg.ENGUID)
		return err
	}

	a.agentsClient.Store(agentType, client)
	zlog.Debug().Msgf("Agent %s - instantiated client to InfrastructureManager: UUID %s", agentType, a.cfg.ENGUID)
	return nil
}

// Connect creates a gRPC Connection to a server.
//
//nolint:gosec // InsecureSkipVerify: true, as we are using CA cert
func connect(
	address, caPath string,
) (*grpc.ClientConn, error) {
	if caPath == "" {
		err := fmt.Errorf("CA cert filepath was not provided")
		zlog.Error().Err(err).Msg("error checking paths of CA cert")
		return nil, err
	}
	creds, err := credentials.NewClientTLSFromFile(caPath, address)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to load CA cert %s", caPath)
		return nil, err
	}

	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
	opts := []grpc.DialOption{
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return net.Dial("tcp", address)
		}),
		grpc.WithTransportCredentials(creds),
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to dial connection to client address %s", address)
		return nil, err
	}
	return conn, nil
}

func (a *Agents) pushStatsEvent(
	source ensimapi.StatusSource,
	mode ensimapi.StatusMode,
	details string,
) {
	event := &ensimapi.NodeStatus{
		Source:  source,
		Mode:    mode,
		Details: details,
	}
	if a.statsChan != nil {
		a.statsChan <- event
	}
}

//nolint:mnd // This function is used to create a host system info message as test example.
func getHostSystemInfoMessage(enUUID, enSerial string) *hostmgr.UpdateHostSystemInfoByGUIDRequest {
	hostSystemInfo := &hostmgr.UpdateHostSystemInfoByGUIDRequest{
		HostGuid: enUUID,
		SystemInfo: &hostmgr.SystemInfo{
			HwInfo: &hostmgr.HWInfo{
				SerialNum: enSerial,
				Cpu: &hostmgr.SystemCPU{
					Cores:   uint32(36),
					Model:   "Intel(R) Xeon(R) Platinum 8360Y CPU @ 2.40GHz",
					Sockets: 2,
					Threads: 144,
					Vendor:  "GenuineIntel",
					Arch:    "x86_64",
				},
				Memory: &hostmgr.SystemMemory{
					Size: 68719476736, // 64GB in bytes
				},
				Gpu: []*hostmgr.SystemGPU{
					{
						PciId:       "0000:00:1f.6",
						Product:     "some product",
						Vendor:      "Vendor",
						Name:        "gpu0",
						Description: "some desc",
					},
					{
						PciId:       "0000:00:1f.7",
						Product:     "some product",
						Vendor:      "Vendor",
						Name:        "gpu1",
						Description: "some desc",
					},
				},
				Storage: &hostmgr.Storage{
					Disk: []*hostmgr.SystemDisk{
						{
							Name:         "sda1",
							SerialNumber: "1234W45678A",
							Vendor:       "Foobar Corp.",
							Model:        "Model",
							Size:         1099511627776, // 1TB in bytes
						},
						{
							Name:         "sda2",
							SerialNumber: "1434W45678B",
							Vendor:       "Foobar Corp.",
							Model:        "Model",
							Size:         1099511627776, // 1TB in bytes
						},
					},
					Features: []string{},
				},
				Usb: []*hostmgr.SystemUSB{
					{
						Bus:      1,
						Addr:     1,
						Class:    "HighFooBar",
						Idvendor: "Foo1",
					},
					{
						Bus:      2,
						Addr:     2,
						Class:    "HighFooBar",
						Idvendor: "Foo2",
					},
				},
				Network: []*hostmgr.SystemNetwork{
					{
						Name:         "ens1",
						Sriovnumvfs:  0,
						Mac:          "90:49:fa:07:6c:fa",
						PciId:        "0000:00:1f.1",
						Sriovenabled: false,
						Mtu:          1500,
						BmcNet:       false,
					},
					{
						Name:         "ens2",
						Sriovnumvfs:  0,
						Mac:          "90:49:fa:07:6c:fb",
						PciId:        "0000:00:1f.2",
						Sriovenabled: false,
						Mtu:          1500,
						BmcNet:       false,
					},
				},
			},
			BiosInfo: &hostmgr.BiosInfo{
				Version:     "1.0.18",
				ReleaseDate: "09/30/2022",
				Vendor:      "Vendor.",
			},
			OsInfo: &hostmgr.OsInfo{},
		},
	}

	return hostSystemInfo
}

func (a *Agents) gatherStatus() (string, bool) {
	counter := 0
	total := 0
	unhealthy := []string{}

	a.settings.Current.Range(func(key, value interface{}) bool {
		total++
		statusValue, okState := value.(AgentState)
		if !okState {
			return false
		}
		statusAgent, okType := key.(AgentType)
		if !okType {
			return false
		}
		// Tolerate 2 missed status messages
		if statusValue == AgentStateOn {
			counter++
		} else { // collect names of agents that are not running
			unhealthy = append(unhealthy, statusAgent.String())
		}
		return true
	})

	zlog.Info().Msgf("%d of %d components running", counter, total)
	if counter != total {
		zlog.Warn().Msgf("Unhealthy components: %v", unhealthy)
	}

	// Return formatted string to HRM, boolean value for instance status
	return fmt.Sprintf("%d of %d components running", counter, total), counter == total
}

func (a *Agents) updateNodeAgent() error {
	zlog.Debug().Msgf("Node Agent - Updating Host Status to InfrastructureManager: UUID %s", a.cfg.ENGUID)

	instanceStatus := hostmgr.InstanceStatus_INSTANCE_STATUS_ERROR
	humanReadableStatus, ok := a.gatherStatus()
	if ok {
		instanceStatus = hostmgr.InstanceStatus_INSTANCE_STATUS_RUNNING
	}

	inUp := &hostmgr.UpdateInstanceStateStatusByHostGUIDRequest{
		HostGuid:             a.cfg.ENGUID,
		InstanceState:        hostmgr.InstanceState_INSTANCE_STATE_RUNNING,
		InstanceStatus:       instanceStatus,
		ProviderStatusDetail: humanReadableStatus,
	}

	authCtx, authErr := utils.GetAuthContext(context.Background(), a.cfg.BaseFolder+defs.NodeAgentTokenPath)
	if authErr != nil {
		zlog.Error().Err(authErr).Msgf("Node Agent - failed to get auth context %s", a.cfg.ENGUID)
		return authErr
	}
	ctx, cancel := context.WithTimeout(authCtx, nodeAgentInterval)
	defer cancel()
	zlog.Debug().Msgf("Node Agent - Update Host Status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	clientValue, ok := a.agentsClient.Load(AgentTypeNode)
	if !ok {
		err := fmt.Errorf("failed to load client HostmgrClient for agent type %s", AgentTypeNode)
		zlog.Error().Err(err).Msgf("Node Agent - failed to cast client to HostmgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	client, ok := clientValue.(hostmgr.HostmgrClient)
	if !ok {
		err := fmt.Errorf("failed to cast client to HostmgrClient")
		zlog.Error().Err(err).Msgf("Node Agent - failed to cast client to HostmgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	_, errUp := client.UpdateInstanceStateStatusByHostGUID(ctx, inUp)
	if errUp != nil {
		zlog.Error().
			Err(errUp).
			Msgf("Node Agent - failed to send UpdateInstanceStateStatusByHostGUID gRPC call: UUID %s", a.cfg.ENGUID)
		a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_NODE_AGENT, ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to send: %s", errUp.Error()))
		return errUp
	}
	a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_NODE_AGENT, ensimapi.StatusMode_STATUS_MODE_OK, "")
	zlog.Debug().Msgf("Node Agent - Successfully Updated Host Status")
	return nil
}

func (a *Agents) updateUpdateAgent() error {
	zlog.Debug().Msgf("Update Agent - Update maintenance status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	inUp := &maintmgr.PlatformUpdateStatusRequest{
		HostGuid: a.cfg.ENGUID,
		UpdateStatus: &maintmgr.UpdateStatus{
			StatusType: maintmgr.UpdateStatus_STATUS_TYPE_UP_TO_DATE,
		},
	}
	upStatusInterval := a.pua.State()
	inUp.UpdateStatus.StatusType = upStatusInterval

	authCtx, authErr := utils.GetAuthContext(context.Background(), a.cfg.BaseFolder+defs.UpdateAgentTokenPath)
	if authErr != nil {
		zlog.Error().Err(authErr).Msgf("Update Agent - failed to get auth context %s", a.cfg.ENGUID)
		return authErr
	}
	ctx, cancel := context.WithTimeout(authCtx, updateAgentInterval)
	defer cancel()

	clientValue, ok := a.agentsClient.Load(AgentTypeUpdate)
	if !ok {
		err := fmt.Errorf("failed to load client to MaintmgrServiceClient for agent type %s", AgentTypeUpdate)
		zlog.Error().Err(err).Msgf("Update Agent - failed to cast client to MaintmgrServiceClient: UUID %s", a.cfg.ENGUID)
		return err
	}

	client, ok := clientValue.(maintmgr.MaintmgrServiceClient)
	if !ok {
		err := fmt.Errorf("failed to cast client to MaintmgrServiceClient")
		zlog.Error().Err(err).Msgf("Update Agent - failed to cast client to MaintmgrServiceClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	resp, err := client.PlatformUpdateStatus(ctx, inUp)
	if err != nil {
		zlog.Error().Err(err).Msgf("Update Agent - failed to send PlatformUpdateStatus gRPC call %s", a.cfg.ENGUID)
		a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_UPDATE_AGENT, ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to send: %s", err.Error()))
		return err
	}
	zlog.Debug().Msgf("Update Agent - Successfully Updated Host Status")
	a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_UPDATE_AGENT, ensimapi.StatusMode_STATUS_MODE_OK,
		inUp.GetUpdateStatus().String())

	a.puaRespChan <- resp
	if upStatusInterval == maintmgr.UpdateStatus_STATUS_TYPE_UPDATED {
		a.pua.UpToDate()
	}

	return nil
}

func (a *Agents) initPUA() error {
	a.pua.Handle(a.wg, a.termChan, a.puaRespChan)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case upStatusAsync := <-a.puaStateChan:
				if a.skipAgentRoutine(AgentTypeUpdate) {
					continue
				}
				zlog.Debug().Msgf("Update Agent - actively update upStatusAsync %v %s", upStatusAsync, a.cfg.ENGUID)
				inUp := &maintmgr.PlatformUpdateStatusRequest{
					HostGuid: a.cfg.ENGUID,
					UpdateStatus: &maintmgr.UpdateStatus{
						StatusType: upStatusAsync,
					},
				}

				authCtx, authErr := utils.GetAuthContext(context.Background(), a.cfg.BaseFolder+defs.UpdateAgentTokenPath)
				if authErr != nil {
					zlog.Error().Err(authErr).Msgf("Update Agent - failed to get auth context %s", a.cfg.ENGUID)
					return
				}
				ctx, cancel := context.WithTimeout(authCtx, updateAgentInterval)

				clientValue, ok := a.agentsClient.Load(AgentTypeUpdate)
				if !ok {
					err := fmt.Errorf("failed to cast client to MaintmgrServiceClient")
					zlog.Error().Err(err).Msgf("Update Agent - failed to cast client to MaintmgrServiceClient: UUID %s",
						a.cfg.ENGUID)
					cancel()
					return
				}
				client, ok := clientValue.(maintmgr.MaintmgrServiceClient)
				if !ok {
					err := fmt.Errorf("failed to cast client to MaintmgrServiceClient")
					zlog.Error().Err(err).Msgf("Update Agent - failed to cast client to MaintmgrServiceClient: UUID %s",
						a.cfg.ENGUID)
					cancel()
					return
				}
				_, errClient := client.PlatformUpdateStatus(ctx, inUp)
				if errClient != nil {
					zlog.Error().Err(errClient).Msgf("Update Agent - failed to send PlatformUpdateStatus gRPC call %s",
						a.cfg.ENGUID)
					a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_UPDATE_AGENT, ensimapi.StatusMode_STATUS_MODE_FAILED,
						fmt.Sprintf("failed to send: %s", errClient.Error()))
					cancel()
					return
				}
				zlog.Debug().Msgf("Update Agent - Successfully Updated Host Status")
				a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_UPDATE_AGENT, ensimapi.StatusMode_STATUS_MODE_OK,
					inUp.GetUpdateStatus().String())
				cancel()

			case <-a.termChan:
				zlog.Debug().Msgf("Agent Update - Terminating PUA routine to InfrastructureManager: UUID %s", a.cfg.ENGUID)
				return
			}
		}
	}()
	return nil
}

func (a *Agents) updateHardwareDiscoveryAgent() error {
	zlog.Debug().Msgf("HD Agent - Update hardware status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	authCtx, authErr := utils.GetAuthContext(context.Background(), a.cfg.BaseFolder+defs.HDAgentTokenPath)
	if authErr != nil {
		zlog.Error().Err(authErr).Msgf("HDA - failed to get auth context %s", a.cfg.ENGUID)
		return authErr
	}
	ctx, cancel := context.WithTimeout(authCtx, hdaInterval)
	defer cancel()
	zlog.Debug().Msgf("HDA - Update Host Status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	clientValue, ok := a.agentsClient.Load(AgentTypeHD)
	if !ok {
		err := fmt.Errorf("failed to load client to HostmgrClient for agent type %s", AgentTypeHD)
		zlog.Error().Err(err).Msgf("HDA - failed to cast client to HostmgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	client, ok := clientValue.(hostmgr.HostmgrClient)
	if !ok {
		err := fmt.Errorf("failed to cast client to HostmgrClient")
		zlog.Error().Err(err).Msgf("HDA - failed to cast client to HostmgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	_, errUp := client.UpdateHostSystemInfoByGUID(ctx, a.hwInfo)
	if errUp != nil {
		zlog.Error().Err(errUp).Msgf("HDA - failed to send UpdateHostStatusByHostGuid gRPC call %s", a.cfg.ENGUID)
		a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_HD_AGENT, ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to send: %s", errUp.Error()))
		return errUp
	}
	zlog.Debug().Msgf("HD Agent - Successfully Updated Host Status")
	a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_HD_AGENT, ensimapi.StatusMode_STATUS_MODE_OK, "")
	return nil
}

func (a *Agents) updateTelemetryAgent() error {
	zlog.Debug().Msgf("Telemetry Agent - Update telemetry status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	inTelMsg := &telmgr.GetTelemetryConfigByGuidRequest{Guid: a.cfg.ENGUID}
	authCtx, authErr := utils.GetAuthContext(context.Background(), a.cfg.BaseFolder+defs.TelemetryAgentTokenPath)
	if authErr != nil {
		zlog.Error().Err(authErr).Msgf("Telemetry Agent - failed to get auth context %s", a.cfg.ENGUID)
		return authErr
	}
	ctx, cancel := context.WithTimeout(authCtx, telemetryAgentInterval)
	defer cancel()

	zlog.Debug().Msgf("Telemetry Agent - Update Host Status to InfrastructureManager: UUID %s", a.cfg.ENGUID)
	clientValue, ok := a.agentsClient.Load(AgentTypeTelemetry)
	if !ok {
		err := fmt.Errorf("failed to load client to TelemetryMgrClient for agent type %s", AgentTypeTelemetry)
		zlog.Error().Err(err).Msgf("Telemetry - failed to cast client to TelemetryMgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	client, ok := clientValue.(telmgr.TelemetryMgrClient)
	if !ok {
		err := fmt.Errorf("failed to cast client to TelemetryMgrClient")
		zlog.Error().Err(err).Msgf("Telemetry - failed to cast client to TelemetryMgrClient: UUID %s", a.cfg.ENGUID)
		return err
	}
	_, errUp := client.GetTelemetryConfigByGUID(ctx, inTelMsg)
	if errUp != nil {
		zlog.Error().
			Err(errUp).
			Msgf("Telemetry Agent - failed to send UpdateHostStatusByHostGuid gRPC call %s", a.cfg.ENGUID)
		a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_TELEMETRY_AGENT, ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to send: %s", errUp.Error()))
		return errUp
	}
	zlog.Debug().Msgf("Telemetry Agent - Successfully Got Config")
	a.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_TELEMETRY_AGENT, ensimapi.StatusMode_STATUS_MODE_OK, "")
	return nil
}

func (a *Agents) setupAgent(agentType AgentType, agentInterval time.Duration, agentUpdateFunc func() error) error {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		zlog.Info().Msgf("Agent %s - Setup routine to InfrastructureManager: UUID %s", agentType, a.cfg.ENGUID)
		err := agentUpdateFunc()
		if err != nil {
			zlog.Warn().Err(err).Msgf("Agent %s - failed to setup initial state: UUID %s", agentType, a.cfg.ENGUID)
		}
		tickerInterval := time.NewTicker(agentInterval)
		defer tickerInterval.Stop()

		tickerConnectionRenew := time.NewTicker(connRenewInterval)
		defer tickerConnectionRenew.Stop()
		for {
			select {
			case <-tickerInterval.C:
				if a.skipAgentRoutine(agentType) {
					continue
				}
				retryErr := backoff.Retry(agentUpdateFunc,
					backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffInterval), backoffRetries))
				if retryErr != nil {
					zlog.Warn().Err(retryErr).Msgf("Agent %s - try reconnect client: UUID %s",
						agentType, a.cfg.ENGUID)
					connErr := a.instantiateClient(agentType, true)
					if connErr != nil {
						zlog.Warn().Msgf("Agent %s - failed to reconnect client to InfrastructureManager: UUID %s",
							agentType, a.cfg.ENGUID)
					}
				}
			case <-tickerConnectionRenew.C:
				connErr := a.instantiateClient(agentType, true)
				if connErr != nil {
					zlog.Warn().Msgf("Agent %s - failed to reconnect client to InfrastructureManager: UUID %s",
						agentType, a.cfg.ENGUID)
				}
			case <-a.termChan:
				zlog.Info().Msgf("Agent %s - Terminating setup routine to InfrastructureManager: UUID %s",
					agentType, a.cfg.ENGUID)
				return
			}
		}
	}()
	return nil
}

func (a *Agents) Start() error {
	err := a.initClients()
	if err != nil {
		zlog.InfraErr(err).Msgf("failed to init agents clients")
		return err
	}

	zlog.Info().Msgf("Simulated agents Starting: UUID %s", a.cfg.ENGUID)

	agentsUpdateMap := map[AgentType]func() error{
		AgentTypeNode:      a.updateNodeAgent,
		AgentTypeHD:        a.updateHardwareDiscoveryAgent,
		AgentTypeUpdate:    a.updateUpdateAgent,
		AgentTypeTelemetry: a.updateTelemetryAgent,
	}

	agentsIntervalMap := map[AgentType]time.Duration{
		AgentTypeNode:      nodeAgentInterval,
		AgentTypeHD:        hdaInterval,
		AgentTypeUpdate:    updateAgentInterval,
		AgentTypeTelemetry: telemetryAgentInterval,
	}

	for agentType, agentUpdateFunc := range agentsUpdateMap {
		setupErr := a.setupAgent(agentType, agentsIntervalMap[agentType], agentUpdateFunc)
		if setupErr != nil {
			zlog.InfraErr(setupErr).Msgf("failed to setup agent %s", agentType)
			return setupErr
		}
	}

	initPUAErr := a.initPUA()
	if initPUAErr != nil {
		zlog.InfraErr(initPUAErr).Msgf("failed to init PUA routine")
		return initPUAErr
	}

	if a.readyChan != nil {
		a.readyChan <- true
	}

	zlog.Info().Msgf("Simulated agents started Successfully: UUID %s", a.cfg.ENGUID)
	return nil
}
