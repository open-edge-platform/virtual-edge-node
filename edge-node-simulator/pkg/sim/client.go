// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
)

type Client interface {
	Create(context.Context, string, *ensimapi.NodeCredentials, bool) error
	CreateNodes(context.Context, uint32, uint32, *ensimapi.NodeCredentials, bool) error
	DeleteNodes(context.Context, uint32) error
	Update(context.Context, string, map[ensimapi.AgentType]ensimapi.AgentState) error
	Get(context.Context, string) (*ensimapi.Node, error)
	List(context.Context) ([]*ensimapi.Node, error)
	Delete(context.Context, string) error
	Close() error
}

type ifmsimClient struct {
	address string
	client  ensimapi.EdgeNodeModelServiceClient
	conn    *grpc.ClientConn
}

func NewClient(ctx context.Context, address string) (Client, error) {
	conn, err := Connect(ctx, address, "", "", "", true)
	if err != nil {
		return nil, err
	}
	grpcClient := ensimapi.NewEdgeNodeModelServiceClient(conn)

	client := &ifmsimClient{
		address: address,
		client:  grpcClient,
		conn:    conn,
	}
	return client, nil
}

func (c *ifmsimClient) Close() error {
	return c.conn.Close()
}

func (c *ifmsimClient) Create(
	ctx context.Context,
	enUUID string,
	enCredentials *ensimapi.NodeCredentials,
	enTeardown bool,
) error {
	zlog.Info().Msg("Create")
	req := &ensimapi.CreateNodeRequest{Uuid: enUUID, Credentials: enCredentials, EnableTeardown: enTeardown}
	_, err := c.client.CreateNode(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to create node %s", enUUID)
		return err
	}
	zlog.Info().Msgf("Created node %s", enUUID)
	return nil
}

func (c *ifmsimClient) CreateNodes(
	ctx context.Context,
	number, batch uint32,
	enCredentials *ensimapi.NodeCredentials,
	enTeardown bool,
) error {
	zlog.Info().Msg("CreateNodes")
	req := &ensimapi.CreateNodesRequest{
		Number:         number,
		BatchSize:      batch,
		Credentials:    enCredentials,
		EnableTeardown: enTeardown,
	}
	resp, err := c.client.CreateNodes(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to create nodes %d", number)
		return err
	}

	nodeUUIDs := []string{}
	for _, node := range resp.Nodes {
		nodeUUIDs = append(nodeUUIDs, node.Uuid)
	}
	zlog.Info().Msgf("Created nodes %d - %v", number, nodeUUIDs)
	return nil
}

func (c *ifmsimClient) Get(ctx context.Context, enUUID string) (*ensimapi.Node, error) {
	zlog.Info().Msg("Get")
	req := &ensimapi.GetNodeRequest{Uuid: enUUID}
	resp, err := c.client.GetNode(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get node %s", enUUID)
		return nil, err
	}
	zlog.Info().Msgf("Got node %s", enUUID)
	node := resp.GetNode()
	return node, nil
}

func (c *ifmsimClient) List(ctx context.Context) ([]*ensimapi.Node, error) {
	zlog.Info().Msg("List")
	nodes := []*ensimapi.Node{}

	req := &ensimapi.ListNodesRequest{}
	resp, err := c.client.ListNodes(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to list nodes")
		return nodes, err
	}

	if resp != nil {
		for {
			respNode, err := resp.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				zlog.Error().Err(err).Msgf("failed to rcv list nodes")
				break
			}
			if respNode != nil {
				nodes = append(nodes, respNode.GetNode())
			}
		}
	}
	zlog.Info().Msgf("Listed nodes %d", len(nodes))
	return nodes, nil
}

func (c *ifmsimClient) Delete(ctx context.Context, enUUID string) error {
	zlog.Info().Msg("Delete")
	req := &ensimapi.DeleteNodeRequest{Uuid: enUUID}
	_, err := c.client.DeleteNode(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to delete node %s", enUUID)
		return err
	}
	zlog.Info().Msgf("Deleted node %s", enUUID)
	return nil
}

func (c *ifmsimClient) DeleteNodes(ctx context.Context, number uint32) error {
	zlog.Info().Msg("DeleteNodes")
	req := &ensimapi.DeleteNodesRequest{Number: number}
	_, err := c.client.DeleteNodes(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to delete nodes %d", number)
		return err
	}
	zlog.Info().Msgf("Deleted nodes %d", number)
	return nil
}

func (c *ifmsimClient) Update(
	ctx context.Context,
	enUUID string,
	agentsStates map[ensimapi.AgentType]ensimapi.AgentState,
) error {
	zlog.Info().Msg("Update")
	agents := []*ensimapi.AgentsStates{}
	for agentType, agentState := range agentsStates {
		agents = append(agents, &ensimapi.AgentsStates{AgentType: agentType, DesiredState: agentState})
	}

	node := &ensimapi.Node{Uuid: enUUID, AgentsStates: agents}
	req := &ensimapi.UpdateNodeRequest{Node: node}
	_, err := c.client.UpdateNode(ctx, req)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to update node %s", enUUID)
	}

	return nil
}
