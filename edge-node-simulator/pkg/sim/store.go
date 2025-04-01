// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// NodeEvent a node event.
type NodeEvent int

const (
	// None non node event.
	None NodeEvent = iota
	// Created created node event.
	Created
	// Updated updated node event.
	Updated
	// Deleted deleted  node event.
	Deleted
)

// String converts node event to string.
func (e NodeEvent) String() string {
	return [...]string{"None", "Created", "Updated", "Deleted"}[e]
}

// Event store event data structure.
type Event struct {
	Key   interface{}
	Value interface{}
	Type  interface{}
}

// NewStore creates a new edge-node store.
func NewStore() Store {
	watchers := NewWatchers()

	return &EdgeNodes{
		edgeNodes: make(map[UUID]*EdgeNode),
		mu:        sync.RWMutex{},
		watchers:  watchers,
	}
}

// Store store interface.
type Store interface {
	// Add   adds the specified edge-node
	Add(en *EdgeNode) error
	// Remove removes the specified edge-node
	Remove(id UUID) error
	// Get gets a edge-node based on a given UUID
	Get(id UUID) (*EdgeNode, error)
	// List lists edgeNodes
	List() ([]*EdgeNode, error)
	// Len number of edgeNodes
	Len() (int, error)

	// Watch watches the node inventory events using the supplied channel
	Watch(ctx context.Context, ch chan<- Event, options ...WatchOptions) error
}

// WatchOptions allows tailoring the WatchNodes behavior.
type WatchOptions struct {
	Replay  bool
	Monitor bool
}

// EdgeNodes data structure for storing EdgeNode.
type EdgeNodes struct {
	edgeNodes map[UUID]*EdgeNode
	mu        sync.RWMutex
	watchers  *Watchers
}

// Len number of edgeNodes.
func (s *EdgeNodes) Len() (int, error) {
	return len(s.edgeNodes), nil
}

// Add adds the specified edge-node.
func (s *EdgeNodes) Add(en *EdgeNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if en.UUID == "" {
		return fmt.Errorf("EdgeNode UUID cannot be empty")
	}
	s.edgeNodes[en.UUID] = en
	addEvent := Event{
		Key:   en.UUID,
		Value: en,
		Type:  Created,
	}
	s.watchers.Send(addEvent)
	return nil
}

// Remove removes the specified edge-node.
func (s *EdgeNodes) Remove(id UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return fmt.Errorf("UUID cannot be empty")
	}
	delete(s.edgeNodes, id)
	delEvent := Event{
		Key:   id,
		Value: nil,
		Type:  Deleted,
	}
	s.watchers.Send(delEvent)

	return nil
}

// Get returns the edge-node with the specified UUID.
func (s *EdgeNodes) Get(id UUID) (*EdgeNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.edgeNodes[id]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("edge-node entry has not been found")
}

// List returns slice containing all current edgeNodes.
func (s *EdgeNodes) List() ([]*EdgeNode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := make([]*EdgeNode, 0, len(s.edgeNodes))
	for _, en := range s.edgeNodes {
		resp = append(resp, en)
	}
	return resp, nil
}

func (s *EdgeNodes) Watch(ctx context.Context, ch chan<- Event, options ...WatchOptions) error {
	replay := len(options) > 0 && options[0].Replay
	id := uuid.New()
	err := s.watchers.AddWatcher(id, ch)
	if err != nil {
		close(ch)
		return err
	}
	go func() {
		<-ctx.Done()
		err = s.watchers.RemoveWatcher(id)
		if err != nil {
			zlog.Error().Err(err).Msgf("failed to remove watcher %s", id)
		}
		close(ch)
	}()

	if replay {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, node := range s.edgeNodes {
				ch <- Event{
					Key:   node.UUID,
					Value: node,
					Type:  None,
				}
			}
		}()
	}
	return nil
}

var _ Store = &EdgeNodes{}
