// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/manifoldco/promptui"
	"golang.org/x/term"
	"google.golang.org/protobuf/encoding/protojson"

	inv_util "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	ensimv1 "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
)

const (
	subPromptSelectSize = 4
	promptSelectSize    = 10
	allowedYN           = "yN"
)

var validateIsConfirm = func(s string) error {
	if strings.Contains(allowedYN, s) && len(s) == 1 {
		return nil
	}
	return fmt.Errorf("invalid input %s of (y/N)", s)
}

func getTerminalSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

func stringContainSearcher(l []menuItem) func(string, int) bool {
	return func(input string, index int) bool {
		return strings.Contains(l[index].Name, input)
	}
}

func validUUID(enUUID string) error {
	if enUUID == "" {
		return fmt.Errorf("node UUID not provided")
	}

	_, err := uuid.Parse(enUUID)
	return err
}

type menuItem struct {
	Name string
	Next func(arg interface{}) (interface{}, error)
	Arg  interface{}
}

var selectTemplate = &promptui.SelectTemplates{
	Label:    "{{ .Name }}?",
	Active:   "▸ {{ .Name | underline }}",
	Inactive: "  {{ .Name }}",
}

var (
	errPromptDone       = fmt.Errorf("prompt done")
	errPromptSelectDone = fmt.Errorf("select done")
	errInvalidAmount    = errors.New("invalid amount")
	errInvalidBatch     = errors.New("invalid batch")
)

func parentPrompt(arg interface{}) (interface{}, error) {
	return arg, errPromptDone
}

func returnSelectedItem(arg interface{}) (interface{}, error) {
	return arg, errPromptSelectDone
}

const (
	labelBack = "<Back>"
)

type CliCfg struct {
	ProjectID       string
	OnboardUsername string
	OnboardPassword string
	APIUsername     string
	APIPassword     string
	EnableNIO       bool
	EnableTeardown  bool
}

type Cli struct {
	ctx              context.Context
	client           Client
	lineItemMaxWidth int
	cfg              *CliCfg
}

func NewCli(ctx context.Context, client Client, cfg *CliCfg) *Cli {
	width, _, err := getTerminalSize()
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed to get terminal size")
	}

	return &Cli{
		ctx:              ctx,
		client:           client,
		lineItemMaxWidth: width - subPromptSelectSize, // subtract the prompt selector.
		cfg:              cfg,
	}
}

func (c *Cli) PromptRoot() (interface{}, error) {
	items := []menuItem{
		{Name: "<Exit>", Next: parentPrompt},
		{Name: "Create Node", Next: c.PromptNodeCreate},
		{Name: "List Nodes", Next: c.PromptNodeList},
		{Name: "Get Node", Next: c.PromptNodeGet},
		{Name: "Delete Node", Next: c.PromptNodeDelete},
		{Name: "Create Nodes", Next: c.PromptNodeCreateMany},
		{Name: "Delete Nodes", Next: c.PromptNodeDeleteMany},
	}
	rootMenuSelect := promptui.Select{
		Label:     "Select Action",
		Items:     items,
		Searcher:  stringContainSearcher(items),
		Templates: selectTemplate,
		Size:      promptSelectSize,
	}
	for {
		i, _, err := rootMenuSelect.Run()
		if err != nil {
			zlog.Error().Err(err).Msg("Prompt failed.")
			return nil, err
		}
		arg, err := items[i].Next(items[i].Arg)
		if errors.Is(err, errPromptDone) {
			return arg, nil
		}
		if errors.Is(err, errPromptSelectDone) {
			continue
		}
		if err != nil {
			return nil, err
		}
	}
}

func (c *Cli) runConfirmPrompt(label, defaultValue string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Default:   defaultValue,
		Validate:  validateIsConfirm,
	}

	result, err := prompt.Run()
	if err != nil && !errors.Is(err, promptui.ErrAbort) {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return false, err
	}
	var value bool
	if result != "y" {
		value = false
	} else {
		value = true
	}
	return value, nil
}

func (c *Cli) runPrompt(label, defaultValue string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
	}
	value, err := prompt.Run()
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return "", err
	}

	return value, nil
}

func (c *Cli) PromptNodeCreate(_ interface{}) (interface{}, error) {
	nodeUUID, err := c.runPrompt("Create Node - UUID (optional)", "")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	enCredentials, err := c.getENCredentials()
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	labelNIO := fmt.Sprintf("Create Node - enable NIO (default: %v)", c.cfg.EnableNIO)
	nodeNIO, err := c.runConfirmPrompt(labelNIO, "N")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	labelTeardown := fmt.Sprintf("Create Node - enable Teardown (default: %v)", c.cfg.EnableTeardown)
	nodeTeardown, err := c.runConfirmPrompt(labelTeardown, "y")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	err = c.client.Create(c.ctx, nodeUUID, enCredentials, nodeNIO, nodeTeardown)
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed to create Node")
	}

	return struct{}{}, nil
}

func (c *Cli) PromptNodeList(_ interface{}) (interface{}, error) {
	for {
		res, err := c.client.List(c.ctx)
		if err != nil {
			return nil, err
		}
		items := []menuItem{
			{Name: labelBack, Next: parentPrompt},
		}
		for _, r := range res {
			items = append(items, menuItem{
				Name: fmt.Sprintf("%.*s", c.lineItemMaxWidth, r.GetUuid()),
				Next: c.PromptNodeGet,
				Arg:  r.GetUuid(),
			})
		}
		prompt := promptui.Select{
			Label:     fmt.Sprintf("%d Nodes total:", len(res)),
			Items:     items,
			Templates: selectTemplate,
			Size:      promptSelectSize,
			Searcher:  stringContainSearcher(items),
		}
		i, _, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return nil, err
		}
		arg, err := items[i].Next(items[i].Arg)
		switch {
		case errors.Is(err, errPromptDone):
			return arg, nil
		case errors.Is(err, errPromptSelectDone):
			return arg, err
		case err != nil:
			return nil, err
		}
	}
}

func (c *Cli) getAndPrintEN(nodeUUID string) error {
	en, err := c.client.Get(c.ctx, nodeUUID)
	if err != nil {
		zlog.Warn().Err(err).Msgf("Failed to retrieve node %s", nodeUUID)
	}
	err = validUUID(nodeUUID)
	if err != nil {
		return err
	}

	// Use protojson to print en in JSON format pretty
	marshaler := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	jsonData, err := marshaler.Marshal(en)
	if err != nil {
		zlog.Error().Err(err).Msg("Failed to marshal node to JSON.")
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func (c *Cli) PromptNodeGet(arg interface{}) (interface{}, error) {
	for {
		var err error
		nodeUUID, ok := arg.(string)
		if !ok {
			nodeUUID, err = c.runPrompt("Get Node by UUID", "")
			if err != nil {
				zlog.Error().Err(err).Msg("Prompt failed.")
				return nil, err
			}
		}

		err = c.getAndPrintEN(nodeUUID)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to get/show edge node")
			return nil, err
		}

		items := []menuItem{
			{Name: labelBack, Next: parentPrompt, Arg: nodeUUID},
			{Name: "<Select>", Next: returnSelectedItem, Arg: nodeUUID},
			{Name: "<Delete>", Next: c.PromptNodeDelete, Arg: nodeUUID},
		}
		promptSelect := promptui.Select{
			Label:     "Node Actions:",
			Items:     items,
			Templates: selectTemplate,
			Size:      promptSelectSize,
		}
		i, _, err := promptSelect.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return nil, err
		}
		arg, err := items[i].Next(items[i].Arg)

		switch {
		case errors.Is(err, errPromptDone):
			return arg, nil
		case errors.Is(err, errPromptSelectDone):
			return arg, err
		case err != nil:
			return nil, err
		}
	}
}

func (c *Cli) PromptNodeDelete(arg interface{}) (interface{}, error) {
	var err error
	nodeUUID, ok := arg.(string)
	if !ok {
		nodeUUID, err = c.runPrompt("Delete Node by UUID", "")
		if err != nil {
			zlog.Error().Err(err).Msg("Prompt failed.")
			return nil, err
		}
	}

	err = validUUID(nodeUUID)
	if err != nil {
		return nil, err
	}

	ack, err := c.runConfirmPrompt("Confirm Node deletion", "y")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	if ack {
		err = c.client.Delete(c.ctx, nodeUUID)
		if err != nil {
			return nil, err
		}
	}
	return struct{}{}, nil
}

func (c *Cli) getENCredentials() (*ensimv1.NodeCredentials, error) {
	labelProjectID := fmt.Sprintf("Create Node - project UUID (default: %s)", c.cfg.ProjectID)
	labelOnbUser := fmt.Sprintf("Create Node - onboard username (default: %s)", c.cfg.OnboardUsername)
	labelOnbPassword := fmt.Sprintf("Create Node - onboard password (default: %s)", c.cfg.OnboardPassword)
	labelAPIUser := fmt.Sprintf("Create Node - api username (default: %s)", c.cfg.APIUsername)
	labelAPIPassword := fmt.Sprintf("Create Node - api password (default: %s)", c.cfg.APIPassword)

	projectUUID, err := c.runPrompt(labelProjectID, c.cfg.ProjectID)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	nodeUserOnb, err := c.runPrompt(labelOnbUser, c.cfg.OnboardUsername)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	nodePasswdOnb, err := c.runPrompt(labelOnbPassword, c.cfg.OnboardPassword)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	nodeUserAPI, err := c.runPrompt(labelAPIUser, c.cfg.APIUsername)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	nodePasswdAPI, err := c.runPrompt(labelAPIPassword, c.cfg.APIPassword)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	enCredentials := &ensimv1.NodeCredentials{
		ProjectId:       projectUUID,
		OnboardUsername: nodeUserOnb,
		OnboardPassword: nodePasswdOnb,
		ApiUsername:     nodeUserAPI,
		ApiPassword:     nodePasswdAPI,
	}
	return enCredentials, nil
}

//nolint:cyclop // this function is a prompt and will be refactored
func (c *Cli) PromptNodeCreateMany(_ interface{}) (interface{}, error) {
	amount, err := c.runPrompt("Create nodes - amount", "1")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	number, err := strconv.Atoi(amount)
	if err != nil {
		zlog.Error().Err(err).Msg("invalid amount")
		return nil, errInvalidAmount
	}

	batchStr, err := c.runPrompt("Create nodes - batch size", "1")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	batch, err := strconv.Atoi(batchStr)
	if err != nil {
		zlog.Error().Err(err).Msg("invalid batch")
		return nil, errInvalidBatch
	}

	unumber, err := inv_util.IntToUint32(number)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	ubatch, err := inv_util.IntToUint32(batch)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	enCredentials, err := c.getENCredentials()
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	labelNIO := fmt.Sprintf("Create Node - enable NIO (default: %v)", c.cfg.EnableNIO)
	nodeNIO, err := c.runConfirmPrompt(labelNIO, "N")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	labelTeardown := fmt.Sprintf("Create Node - enable Teardown (default: %v)", c.cfg.EnableTeardown)
	nodeTeardown, err := c.runConfirmPrompt(labelTeardown, "y")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	ack, err := c.runConfirmPrompt("Confirm Nodes creation", "y")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	if ack {
		err = c.client.CreateNodes(c.ctx, unumber, ubatch, enCredentials, nodeNIO, nodeTeardown)
		if err != nil {
			return nil, err
		}
	}
	return struct{}{}, nil
}

func (c *Cli) PromptNodeDeleteMany(arg interface{}) (interface{}, error) {
	number, ok := arg.(int)
	if !ok {
		amount, err := c.runPrompt("Delete nodes - amount", "1")
		if err != nil {
			zlog.Error().Err(err).Msg("Prompt failed.")
			return nil, err
		}

		number, err = strconv.Atoi(amount)
		if err != nil {
			zlog.Error().Err(err).Msg("invalid amount")
			return nil, errInvalidAmount
		}
	}

	unumber, err := inv_util.IntToUint32(number)
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}

	ack, err := c.runConfirmPrompt("Confirm Nodes delete", "N")
	if err != nil {
		zlog.Error().Err(err).Msg("Prompt failed.")
		return nil, err
	}
	if ack {
		err = c.client.DeleteNodes(c.ctx, unumber)
		if err != nil {
			return nil, err
		}
	}
	return struct{}{}, nil
}
