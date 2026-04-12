package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// externalRequest is the JSON envelope sent to an external plugin binary via stdin.
type externalRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params,omitempty"`
}

// externalResponse is the JSON envelope received from an external plugin binary via stdout.
type externalResponse struct {
	OK     bool   `json:"ok"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// pluginTimeout is the maximum time a plugin binary may run before being killed.
const pluginTimeout = 30 * time.Second

// callPlugin executes the plugin binary with a JSON request on stdin and reads the response.
func callPlugin(entrypoint string, req externalRequest) (*externalResponse, error) {
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal plugin request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, entrypoint)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plugin %s failed: %w\nstderr: %s", entrypoint, err, stderr.String())
	}

	var resp externalResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("plugin %s: invalid response: %w", entrypoint, err)
	}

	if !resp.OK {
		return nil, fmt.Errorf("plugin %s: %s", entrypoint, resp.Error)
	}

	return &resp, nil
}

// externalProvider wraps an external binary as a Provider.
type externalProvider struct {
	manifest   Manifest
	entrypoint string
}

// NewExternalProvider creates a Provider that delegates to an external binary.
func NewExternalProvider(m Manifest, entrypoint string) Provider {
	return &externalProvider{manifest: m, entrypoint: entrypoint}
}

func (p *externalProvider) Create(name string, image string, opts CreateOpts) error {
	params := map[string]any{
		"name":  name,
		"image": image,
	}
	if len(opts.Ports) > 0 {
		params["ports"] = opts.Ports
	}
	if len(opts.Env) > 0 {
		params["env"] = opts.Env
	}
	if opts.CPUs > 0 {
		params["cpus"] = opts.CPUs
	}
	if opts.Memory != "" {
		params["memory"] = opts.Memory
	}
	if len(opts.Volumes) > 0 {
		params["volumes"] = opts.Volumes
	}
	if opts.Command != "" {
		params["command"] = opts.Command
	}
	_, err := callPlugin(p.entrypoint, externalRequest{Action: "create", Params: params})
	return err
}

func (p *externalProvider) Start(name string) error {
	_, err := callPlugin(p.entrypoint, externalRequest{
		Action: "start",
		Params: map[string]any{"name": name},
	})
	return err
}

func (p *externalProvider) Stop(name string) error {
	_, err := callPlugin(p.entrypoint, externalRequest{
		Action: "stop",
		Params: map[string]any{"name": name},
	})
	return err
}

func (p *externalProvider) Destroy(name string) error {
	_, err := callPlugin(p.entrypoint, externalRequest{
		Action: "destroy",
		Params: map[string]any{"name": name},
	})
	return err
}

func (p *externalProvider) Status(name string) (string, error) {
	resp, err := callPlugin(p.entrypoint, externalRequest{
		Action: "status",
		Params: map[string]any{"name": name},
	})
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

func (p *externalProvider) ProviderName() string {
	return p.manifest.Name
}

// externalHook wraps an external binary as a Hook.
type externalHook struct {
	manifest   Manifest
	entrypoint string
}

// NewExternalHook creates a Hook that delegates to an external binary.
func NewExternalHook(m Manifest, entrypoint string) Hook {
	return &externalHook{manifest: m, entrypoint: entrypoint}
}

func (h *externalHook) Execute(ctx HookContext) error {
	params := map[string]any{
		"workspace": ctx.WorkspaceName,
		"event":     string(ctx.Event),
		"server":    ctx.ServerHost,
	}
	if len(ctx.Env) > 0 {
		params["env"] = ctx.Env
	}
	_, err := callPlugin(h.entrypoint, externalRequest{Action: "execute", Params: params})
	return err
}

func (h *externalHook) Events() []Event {
	return h.manifest.Events
}

func (h *externalHook) HookName() string {
	return h.manifest.Name
}
