package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type ProviderUsage struct {
	Kind    string        `json:"kind"`
	Plan    string        `json:"plan,omitempty"`
	Windows []UsageWindow `json:"windows"`
}

type UsageWindow struct {
	UsedPercent     int        `json:"usedPercent"`
	DurationMinutes int64      `json:"durationMinutes,omitempty"`
	ResetsAt        *time.Time `json:"resetsAt,omitempty"`
}

type codexRPCRequest struct {
	ID     int    `json:"id,omitempty"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type codexRPCResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *codexRPCError  `json:"error"`
}

type codexRPCError struct {
	Message string `json:"message"`
}

type codexRateLimitsResponse struct {
	RateLimits struct {
		PlanType  string            `json:"planType"`
		Primary   *codexUsageWindow `json:"primary"`
		Secondary *codexUsageWindow `json:"secondary"`
	} `json:"rateLimits"`
}

type codexUsageWindow struct {
	UsedPercent        int    `json:"usedPercent"`
	WindowDurationMins *int64 `json:"windowDurationMins"`
	ResetsAt           *int64 `json:"resetsAt"`
}

func ReadUsage(ctx context.Context) ([]ProviderUsage, error) {
	usage, err := readCodexUsage(ctx)
	if errors.Is(err, exec.ErrNotFound) {
		return []ProviderUsage{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(usage.Windows) == 0 {
		return []ProviderUsage{}, nil
	}
	return []ProviderUsage{usage}, nil
}

func readCodexUsage(ctx context.Context) (ProviderUsage, error) {
	command := exec.CommandContext(ctx, "codex", "app-server", "--stdio")
	stdin, err := command.StdinPipe()
	if err != nil {
		return ProviderUsage{}, fmt.Errorf("open Codex app-server input: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return ProviderUsage{}, fmt.Errorf("open Codex app-server output: %w", err)
	}
	if err := command.Start(); err != nil {
		return ProviderUsage{}, fmt.Errorf("start Codex app server: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)
	if err := encoder.Encode(codexRPCRequest{
		ID:     1,
		Method: "initialize",
		Params: map[string]any{
			"clientInfo": map[string]string{
				"name":    "pickle",
				"title":   "Pickle",
				"version": "0.0.0",
			},
			"capabilities": map[string]bool{"experimentalApi": true},
		},
	}); err != nil {
		return ProviderUsage{}, fmt.Errorf("initialize Codex app server: %w", err)
	}
	if _, err := readCodexResponse(decoder, 1); err != nil {
		return ProviderUsage{}, fmt.Errorf("initialize Codex app server: %w", err)
	}
	if err := encoder.Encode(codexRPCRequest{Method: "initialized"}); err != nil {
		return ProviderUsage{}, fmt.Errorf("acknowledge Codex app server: %w", err)
	}
	if err := encoder.Encode(codexRPCRequest{ID: 2, Method: "account/rateLimits/read"}); err != nil {
		return ProviderUsage{}, fmt.Errorf("request Codex usage: %w", err)
	}
	response, err := readCodexResponse(decoder, 2)
	if err != nil {
		return ProviderUsage{}, fmt.Errorf("read Codex usage: %w", err)
	}

	return parseCodexUsage(response.Result)
}

func readCodexResponse(decoder *json.Decoder, id int) (codexRPCResponse, error) {
	for {
		var response codexRPCResponse
		if err := decoder.Decode(&response); err != nil {
			if errors.Is(err, io.EOF) {
				return codexRPCResponse{}, errors.New("Codex app server closed unexpectedly")
			}
			return codexRPCResponse{}, err
		}
		if response.ID != id {
			continue
		}
		if response.Error != nil {
			return codexRPCResponse{}, errors.New(response.Error.Message)
		}
		return response, nil
	}
}

func parseCodexUsage(data []byte) (ProviderUsage, error) {
	var response codexRateLimitsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return ProviderUsage{}, fmt.Errorf("decode Codex usage: %w", err)
	}

	usage := ProviderUsage{Kind: "codex", Plan: response.RateLimits.PlanType, Windows: []UsageWindow{}}
	for _, window := range []*codexUsageWindow{response.RateLimits.Primary, response.RateLimits.Secondary} {
		if window == nil {
			continue
		}
		parsed := UsageWindow{UsedPercent: window.UsedPercent}
		if window.WindowDurationMins != nil {
			parsed.DurationMinutes = *window.WindowDurationMins
		}
		if window.ResetsAt != nil {
			reset := time.Unix(*window.ResetsAt, 0).UTC()
			parsed.ResetsAt = &reset
		}
		usage.Windows = append(usage.Windows, parsed)
	}
	return usage, nil
}
