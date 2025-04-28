// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim_agents "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/agents"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	deleteBatchSize = 100
)

// NewService returns a new model Service.
func NewIFMSimService(nodeStore Store, cfg *Config) northbound.Service {
	return &Service{
		nodeStore: nodeStore,
		cfg:       cfg,
	}
}

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
	nodeStore Store
	cfg       *Config
}

// Register registers the TrafficSim Service with the gRPC server.
func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		nodeStore: s.nodeStore,
		cfg:       s.cfg,
	}
	ensimapi.RegisterEdgeNodeModelServiceServer(r, server)
	reflection.Register(r)
}

// Server implements the TrafficSim gRPC service for administrative facilities.
type Server struct {
	nodeStore Store
	cfg       *Config
}

// NewUUID returns the locally unique UUID for the edge node.
func NewUUID(enUUID string) (string, error) {
	if enUUID != "" {
		_, err := uuid.Parse(enUUID)
		if err != nil {
			return "", err
		}
		return enUUID, nil
	}
	newUUID := uuid.New().String()
	zlog.Info().Msgf("UUID not provided, creating one: %s", newUUID)
	return newUUID, nil
}

func (s *Server) buildENSettings(enUUID, enProject,
	enUser, enPasswd, enAPIUser, enAPIPasswd string,
	enableNIO, enableTeardown bool,
) (*defs.Settings, error) {
	enUUID, err := NewUUID(enUUID)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate %s", enUUID)
		return nil, err
	}

	certCA, err := LoadFile(s.cfg.OrchCAPath)
	if err != nil {
		zlog.Err(err).Msg("failed to read certCA config file")
		return nil, err
	}

	macAddress, err := utils.GetRandomMACAddress()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get random MAC address")
		return nil, err
	}

	enBasePath := s.cfg.BaseFolder + "/" + enUUID
	enSerial := strings.ReplaceAll(enUUID, "-", "")[:20]

	setting := &defs.Settings{
		OrchFQDN:        s.cfg.OrchFQDN,
		ENGUID:          enUUID,
		ENSerial:        enSerial,
		EdgeOnboardUser: enUser,
		EdgeOnboardPass: enPasswd,
		EdgeAPIUser:     enAPIUser,
		EdgeAPIPass:     enAPIPasswd,
		CertCA:          certCA,
		CertCAPath:      s.cfg.OrchCAPath,
		RunAgents:       true,
		NIOnboard:       enableNIO,
		SouthOnboard:    !enableNIO,
		SetupTeardown:   enableTeardown,
		Project:         enProject,
		Org:             "",
		BaseFolder:      enBasePath,
		MACAddress:      macAddress,
		AutoProvision:   true,
	}
	zlog.Info().Msgf("Node settings %v", setting)
	return setting, nil
}

func (s *Server) CreateNodes(
	_ context.Context,
	req *ensimapi.CreateNodesRequest,
) (*ensimapi.CreateNodesResponse, error) {
	zlog.Info().Msgf("CreateNodes %d by batch %d with credentials %v",
		req.GetNumber(), req.GetBatchSize(), req.GetCredentials())

	err := req.ValidateAll()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate create nodes request")
		return nil, err
	}
	total := int(req.GetNumber())
	batch := int(req.GetBatchSize())
	if batch == 0 {
		batch = 1
	}

	if batch > total {
		zlog.Info().Msgf("Create Nodes batch %d bigger than total %d - fixing it to equal to total", batch, total)
		batch = total
	}

	enUUIDs := make(chan string, total)
	errChan := make(chan error, total)

	wg := &sync.WaitGroup{}
	for num := 1; num <= total; num++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			zlog.Info().Msgf("Create Node %d out of %d", id, total)
			enUUID, errUUID := s.helperCreateNode("",
				req.GetCredentials().GetProject(),
				req.GetCredentials().GetOnboardUsername(),
				req.GetCredentials().GetOnboardPassword(),
				req.GetCredentials().GetApiUsername(),
				req.GetCredentials().GetApiPassword(),
				req.GetEnableNio(),
				req.GetEnableTeardown(),
			)
			if errUUID != nil {
				zlog.Error().Err(errUUID).Msgf("failed to create node %d %s", id, enUUID)
				errChan <- errUUID
			} else {
				enUUIDs <- enUUID
			}
		}(num)

		if num%batch == 0 {
			zlog.Info().Msgf("Created Node batch %d out of %d", num%batch, total/batch)
			wg.Wait()
		}
	}
	wg.Wait()
	close(errChan)
	close(enUUIDs)

	errAll := CheckErrors(errChan)
	if errAll != nil {
		zlog.Error().Err(err).Msgf("failed to create nodes")
		return nil, errAll
	}

	nodes := []*ensimapi.Node{}
	for enUUID := range enUUIDs {
		node := &ensimapi.Node{
			Uuid: enUUID,
		}
		nodes = append(nodes, node)
	}

	resp := &ensimapi.CreateNodesResponse{
		Nodes: nodes,
	}
	return resp, nil
}

func (s *Server) CreateNode(
	_ context.Context,
	req *ensimapi.CreateNodeRequest,
) (*ensimapi.CreateNodeResponse, error) {
	err := req.ValidateAll()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate create node request")
		return nil, err
	}
	zlog.Info().Msgf("CreateNode %s with credentials %v",
		req.GetUuid(), req.GetCredentials())

	_, err = s.helperCreateNode(req.GetUuid(),
		req.GetCredentials().GetProject(),
		req.GetCredentials().GetOnboardUsername(),
		req.GetCredentials().GetOnboardPassword(),
		req.GetCredentials().GetApiUsername(),
		req.GetCredentials().GetApiPassword(),
		req.GetEnableNio(),
		req.GetEnableTeardown(),
	)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to create node %s", req.GetUuid())
		return nil, err
	}

	return &ensimapi.CreateNodeResponse{}, nil
}

func helperAgentState(state ensim_agents.AgentState) ensimapi.AgentState {
	statesMap := map[ensim_agents.AgentState]ensimapi.AgentState{
		ensim_agents.AgentStateOn:  ensimapi.AgentState_AGENT_STATE_ON,
		ensim_agents.AgentStateOff: ensimapi.AgentState_AGENT_STATE_OFF,
	}
	return statesMap[state]
}

func helperAgentType(agentType ensim_agents.AgentType) ensimapi.AgentType {
	typesMap := map[ensim_agents.AgentType]ensimapi.AgentType{
		ensim_agents.AgentTypeNode:      ensimapi.AgentType_AGENT_TYPE_NODE,
		ensim_agents.AgentTypeHD:        ensimapi.AgentType_AGENT_TYPE_HD,
		ensim_agents.AgentTypeUpdate:    ensimapi.AgentType_AGENT_TYPE_UPDATE,
		ensim_agents.AgentTypeTelemetry: ensimapi.AgentType_AGENT_TYPE_TELEMETRY,
	}
	return typesMap[agentType]
}

func helperNodeStates(existingStates ensim_agents.StateMap) []*ensimapi.AgentsStates {
	agents := []*ensimapi.AgentsStates{}
	for agentType, currentState := range existingStates.Current {
		agent := &ensimapi.AgentsStates{
			AgentType:    helperAgentType(agentType),
			CurrentState: helperAgentState(currentState),
			DesiredState: helperAgentState(existingStates.Desired[agentType]),
		}
		agents = append(agents, agent)
	}

	return agents
}

func enToProto(en *EdgeNode) *ensimapi.Node {
	enStatus := []*ensimapi.NodeStatus{}
	for _, stats := range en.GetAgentsStatus() {
		enStatus = append(enStatus, stats)
	}

	agentsStates := en.GetAgentsStates()
	agents := helperNodeStates(agentsStates)
	return &ensimapi.Node{
		Uuid:         string(en.UUID),
		Status:       enStatus,
		AgentsStates: agents,
		EnableNio:    en.cfg.NIOnboard,
		Credentials: &ensimapi.NodeCredentials{
			Project:         en.cfg.Project,
			OnboardUsername: en.cfg.EdgeOnboardUser,
			OnboardPassword: en.cfg.EdgeOnboardPass,
			ApiUsername:     en.cfg.EdgeAPIUser,
			ApiPassword:     en.cfg.EdgeAPIPass,
		},
	}
}

func (s *Server) GetNode(_ context.Context, req *ensimapi.GetNodeRequest) (*ensimapi.GetNodeResponse, error) {
	zlog.Info().Msgf("GetNode %s", req.GetUuid())
	err := req.ValidateAll()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate get node request")
		return nil, err
	}
	en, err := s.nodeStore.Get(UUID(req.GetUuid()))
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get in store %s", req.GetUuid())
		return nil, err
	}
	zlog.Info().Msgf("Store got node %s", req.GetUuid())
	enProto := enToProto(en)
	return &ensimapi.GetNodeResponse{
		Node: enProto,
	}, nil
}

func (s *Server) DeleteNode(
	_ context.Context,
	req *ensimapi.DeleteNodeRequest,
) (*ensimapi.DeleteNodeResponse, error) {
	zlog.Info().Msgf("DeleteNode %s", req.GetUuid())
	err := req.ValidateAll()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate delete node request")
		return nil, err
	}
	err = s.helperDeleteNode(req.GetUuid())
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to delete %s", req.GetUuid())
		return nil, err
	}

	return &ensimapi.DeleteNodeResponse{}, nil
}

func (s *Server) WatchNodes(req *ensimapi.WatchNodesRequest, serv ensimapi.EdgeNodeModelService_WatchNodesServer) error {
	zlog.Info().Msgf("WatchNodes")
	ch := make(chan Event)
	err := s.nodeStore.Watch(serv.Context(), ch, WatchOptions{Replay: !req.NoReplay, Monitor: !req.NoSubscribe})
	if err != nil {
		return err
	}

	for nodeEvent := range ch {
		en, ok := nodeEvent.Value.(*EdgeNode)
		if !ok {
			errFmt := fmt.Errorf("failed to parse event value")
			zlog.Error().Err(err).Msgf("failed to convert to EdgeNode")
			return errFmt
		}
		node := enToProto(en)

		evType, ok := nodeEvent.Type.(ensimapi.EventType)
		if !ok {
			errFmt := fmt.Errorf("failed to parse event type")
			zlog.Error().Err(err).Msgf("failed to convert to EventType")
			return errFmt
		}
		response := &ensimapi.WatchNodesResponse{
			Node: node,
			Type: evType,
		}
		err := serv.Send(response)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) ListNodes(_ *ensimapi.ListNodesRequest, serv ensimapi.EdgeNodeModelService_ListNodesServer) error {
	zlog.Info().Msgf("ListNodes")

	enList, err := s.nodeStore.List()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to list in store")
		return err
	}

	for _, en := range enList {
		node := enToProto(en)
		err = serv.Send(&ensimapi.ListNodesResponse{
			Node: node,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) DeleteNodes(
	_ context.Context,
	req *ensimapi.DeleteNodesRequest,
) (*ensimapi.DeleteNodesResponse, error) {
	zlog.Info().Msgf("DeleteNodes %d", req.GetNumber())

	enList, err := s.nodeStore.List()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to list in store")
		return nil, err
	}

	if int(req.GetNumber()) > len(enList) {
		err = fmt.Errorf("requested amount of delete nodes is bigger than existing number")
		zlog.Error().Err(err).Msgf("failed to validate delete number")
		return nil, err
	}

	totalDeleteNodes := int(req.GetNumber())
	if totalDeleteNodes == 0 {
		totalDeleteNodes = len(enList)
	}

	numDeletedNodes := 0
	wg := &sync.WaitGroup{}
	errChan := make(chan error, len(enList))

	for _, en := range enList {
		wg.Add(1)
		go func(enUUID string) {
			defer wg.Done()
			errDel := s.helperDeleteNode(enUUID)
			if errDel != nil {
				zlog.Error().Err(errDel).Msgf("failed to delete %s", enUUID)
				errChan <- errDel
			}
		}(string(en.UUID))

		numDeletedNodes++
		zlog.Info().Msgf("Deleted nodes %d out of %d", numDeletedNodes, totalDeleteNodes)
		if numDeletedNodes >= totalDeleteNodes {
			break
		}

		if numDeletedNodes%deleteBatchSize == 0 {
			wg.Wait()
		}
	}
	wg.Wait()
	close(errChan)

	errAll := CheckErrors(errChan)
	if errAll != nil {
		zlog.Error().Err(errAll).Msgf("failed to delete nodes")
		return nil, errAll
	}

	zlog.Info().Msgf("Deleted nodes %d", totalDeleteNodes)
	return &ensimapi.DeleteNodesResponse{}, nil
}

func (s *Server) helperCreateNode(enUUID, enProject,
	enUser, enPasswd, enAPIUser, enAPIPasswd string,
	enableNIO, enableTeardown bool,
) (string, error) {
	enCfg, err := s.buildENSettings(enUUID, enProject,
		enUser, enPasswd, enAPIUser, enAPIPasswd,
		enableNIO, enableTeardown,
	)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to build edge node settings")
		return "", err
	}

	en := NewEdgeNode(enCfg)
	err = en.Start()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to start %s", enCfg.ENGUID)
		en.Stop()
		return enCfg.ENGUID, err
	}

	zlog.Info().Msgf("Store adding node %s", en.cfg.ENGUID)
	err = s.nodeStore.Add(en)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to store %s", en.cfg.ENGUID)
		en.Stop()
		return enCfg.ENGUID, err
	}

	return enCfg.ENGUID, nil
}

func (s *Server) helperDeleteNode(enUUID string) error {
	zlog.Info().Msgf("DeleteNode %s", enUUID)

	en, err := s.nodeStore.Get(UUID(enUUID))
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get in store %s", enUUID)
		return err
	}

	zlog.Info().Msgf("Stopping Node %s", enUUID)
	err = en.Stop()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to stop %s", enUUID)
		return err
	}

	zlog.Info().Msgf("Store removing Node %s", enUUID)
	err = s.nodeStore.Remove(UUID(enUUID))
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to delete in store %s", enUUID)
		return err
	}

	return nil
}

func helperUpdateNode(currentNode *EdgeNode, node *ensimapi.Node) error {
	if node.GetUuid() != currentNode.cfg.ENGUID {
		return fmt.Errorf("node UUID %s is different from %s", node.GetUuid(), currentNode.cfg.ENGUID)
	}

	isAgentAlive := func(state *ensimapi.AgentsStates) bool {
		return state.DesiredState == ensimapi.AgentState_AGENT_STATE_ON
	}

	updateAgentSetting := func(
		alive bool,
		desiredStatus map[ensim_agents.AgentType]ensim_agents.AgentState,
		agentsStates *ensimapi.AgentsStates,
	) {
		agentMode := ensim_agents.AgentStateOff
		if alive {
			agentMode = ensim_agents.AgentStateOn
		}

		switch agentsStates.AgentType {
		case ensimapi.AgentType_AGENT_TYPE_NODE:
			desiredStatus[ensim_agents.AgentTypeNode] = agentMode
		case ensimapi.AgentType_AGENT_TYPE_HD:
			desiredStatus[ensim_agents.AgentTypeHD] = agentMode
		case ensimapi.AgentType_AGENT_TYPE_UPDATE:
			desiredStatus[ensim_agents.AgentTypeUpdate] = agentMode
		case ensimapi.AgentType_AGENT_TYPE_TELEMETRY:
			desiredStatus[ensim_agents.AgentTypeTelemetry] = agentMode
		case ensimapi.AgentType_AGENT_TYPE_UNSPECIFIED:
			zlog.Warn().Msgf("Unknown agent type %s", agentsStates.AgentType)
			return
		}
	}

	agentsDesiredStatus := map[ensim_agents.AgentType]ensim_agents.AgentState{}
	for _, agentState := range node.GetAgentsStates() {
		agentAlive := isAgentAlive(agentState)
		updateAgentSetting(agentAlive, agentsDesiredStatus, agentState)
	}

	currentNode.SetAgentsStates(agentsDesiredStatus)
	return nil
}

func (s *Server) UpdateNode(
	_ context.Context,
	req *ensimapi.UpdateNodeRequest,
) (*ensimapi.UpdateNodeResponse, error) {
	node := req.GetNode()
	zlog.Info().Msgf("UpdateNode %s", node.String())
	err := req.ValidateAll()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to validate update node request")
		return nil, err
	}

	nodeUUID := node.GetUuid()
	currentNode, err := s.nodeStore.Get(UUID(node.GetUuid()))
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get in store %s", nodeUUID)
		return nil, err
	}

	err = helperUpdateNode(currentNode, node)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to update %s", node.GetUuid())
		return nil, err
	}

	zlog.Info().Msgf("Updated node %s", node.GetUuid())
	return &ensimapi.UpdateNodeResponse{}, nil
}

func CheckErrors(errChan chan error) error {
	allErrors := []string{}
	for errDel := range errChan {
		if errDel != nil {
			allErrors = append(allErrors, errDel.Error())
		}
	}
	if len(allErrors) > 0 {
		errAllDel := fmt.Errorf("%v", allErrors)
		return errAllDel
	}
	return nil
}
