package autohand

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Transport manages the CLI subprocess and JSON-RPC communication.
type Transport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	lineReader *LineReader

	mu        sync.Mutex
	callbacks map[int]chan transportResponse
	notify    map[string]func(json.RawMessage)
	nextID    int
	debug     bool
	timeout   time.Duration
}

// NewTransport creates a Transport with the given config.
func NewTransport(cfg *Config) *Transport {
	timeout := 300 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}
	return &Transport{
		callbacks: make(map[int]chan transportResponse),
		notify:    make(map[string]func(json.RawMessage)),
		debug:     cfg.Debug,
		timeout:   timeout,
	}
}

// Start spawns the CLI subprocess.
func (t *Transport) Start(ctx context.Context, cfg *Config) error {
	cliPath := cfg.CLIPath
	if cliPath == "" {
		var err error
		cliPath, err = t.detectCLIBinary()
		if err != nil {
			return fmt.Errorf("detect CLI binary: %w", err)
		}
	}

	cwd := cfg.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}

	// Copy custom skill files before starting CLI
	if err := t.copySkillFiles(cfg, cwd); err != nil {
		t.log("copySkillFiles warning: %s", err)
	}

	args := []string{"--mode", "rpc"}

	if cfg.Unrestricted {
		args = append(args, "--unrestricted")
	}
	if cfg.AutoMode {
		args = append(args, "--auto-mode")
	}
	if cfg.AutoSkill {
		args = append(args, "--auto-skill")
	}
	if cfg.AutoCommit {
		args = append(args, "-c")
	}
	if cfg.ContextCompact {
		args = append(args, "--context-compact")
	}
	if cfg.PersistSession {
		args = append(args, "--persist-session")
	}
	if cfg.SessionID != "" {
		args = append(args, "--session-id", cfg.SessionID)
	}
	if cfg.Resume {
		args = append(args, "--resume")
	}
	if cfg.Continue {
		args = append(args, "--continue")
	}
	if cfg.SessionPath != "" {
		args = append(args, "--session-path", cfg.SessionPath)
	}
	if cfg.AutoSaveInterval > 0 {
		args = append(args, "--auto-save-interval", fmt.Sprintf("%d", cfg.AutoSaveInterval))
	}
	if cfg.AgentsMdEnable {
		args = append(args, "--agents-md")
	}
	if cfg.AgentsMdCreate {
		args = append(args, "--agents-md-create")
	}
	if cfg.AgentsMdPath != "" {
		args = append(args, "--agents-md-path", cfg.AgentsMdPath)
	}
	if cfg.AgentsMdAutoUpdate {
		args = append(args, "--agents-md-auto-update")
	}
	if cfg.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", cfg.MaxTokens))
	}
	if cfg.CompressionThreshold > 0 {
		args = append(args, "--compression-threshold", fmt.Sprintf("%f", cfg.CompressionThreshold))
	}
	if cfg.SummarizationThreshold > 0 {
		args = append(args, "--summarization-threshold", fmt.Sprintf("%f", cfg.SummarizationThreshold))
	}
	if len(cfg.Skills) > 0 {
		names := make([]string, len(cfg.Skills))
		for i, s := range cfg.Skills {
			names[i] = s.Name
		}
		args = append(args, "--skills", strings.Join(names, ","))
	}
	if cfg.MaxIterations > 0 {
		args = append(args, "--max-iterations", fmt.Sprintf("%d", cfg.MaxIterations))
	}
	if cfg.MaxRuntime > 0 {
		args = append(args, "--max-runtime", fmt.Sprintf("%d", cfg.MaxRuntime))
	}
	if cfg.MaxCost > 0 {
		args = append(args, "--max-cost", fmt.Sprintf("%f", cfg.MaxCost))
	}
	if cfg.SysPrompt != "" {
		args = append(args, "--sys-prompt", cfg.SysPrompt)
	}
	if cfg.AppendSysPrompt != "" {
		args = append(args, "--append-sys-prompt", cfg.AppendSysPrompt)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.Temperature > 0 {
		args = append(args, "--temperature", fmt.Sprintf("%f", cfg.Temperature))
	}
	if cfg.Yolo != "" {
		args = append(args, "--yolo", cfg.Yolo)
	}
	if cfg.YoloTimeout > 0 {
		args = append(args, "--yolo-timeout", fmt.Sprintf("%d", cfg.YoloTimeout))
	}
	for _, dir := range cfg.AddDir {
		args = append(args, "--add-dir", dir)
	}
	for _, dir := range cfg.AdditionalDirectories {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, cfg.ExtraArgs...)

	t.log("Starting CLI: %s", cliPath)
	t.log("Args: %v", args)

	env := os.Environ()
	env = append(env, "AUTOHAND_STREAM_TOOL_OUTPUT=1")
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range cfg.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	t.cmd = exec.CommandContext(ctx, cliPath, args...)
	t.cmd.Dir = cwd
	t.cmd.Env = env

	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("start CLI: %w", err)
	}

	t.lineReader = NewLineReader(t.stdout)
	go t.readLoop()

	go func() {
		data, _ := io.ReadAll(t.stderr)
		if len(data) > 0 {
			t.log("STDERR: %s", string(data))
		}
	}()

	time.Sleep(500 * time.Millisecond)
	return nil
}

// Stop terminates the CLI subprocess.
func (t *Transport) Stop() error {
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	t.log("Stopping CLI process")
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.lineReader != nil {
		t.lineReader.Close()
	}
	if err := t.cmd.Process.Signal(os.Interrupt); err != nil {
		t.cmd.Process.Kill()
	}
	t.cmd.Wait()
	return nil
}

// Request sends a JSON-RPC request and waits for the response.
func (t *Transport) Request(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	t.mu.Lock()
	id := t.nextID
	t.nextID++
	ch := make(chan transportResponse, 1)
	t.callbacks[id] = ch
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.callbacks, id)
		t.mu.Unlock()
	}()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	t.log("Sending request: %s (id: %d)", method, id)

	if _, err := t.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("RPC error: %s", resp.err.Error())
		}
		return resp.result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(t.timeout):
		return nil, fmt.Errorf("request timeout: %s", method)
	}
}

// OnNotification registers a notification handler.
func (t *Transport) OnNotification(method string, handler func(json.RawMessage)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.notify[method] = handler
}

// OffNotification removes a notification handler.
func (t *Transport) OffNotification(method string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.notify, method)
}

// IsRunning returns whether the process is running.
func (t *Transport) IsRunning() bool {
	return t.cmd != nil && t.cmd.Process != nil
}

func (t *Transport) readLoop() {
	for {
		line, err := t.lineReader.ReadLine()
		if err != nil {
			return
		}
		t.handleLine(line)
	}
}

func (t *Transport) handleLine(line string) {
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.log("Error parsing line: %s", line)
		return
	}

	if resp.ID != 0 {
		t.mu.Lock()
		ch, ok := t.callbacks[resp.ID]
		t.mu.Unlock()
		if ok {
			ch <- transportResponse{result: resp.Result, err: resp.Error}
		}
		return
	}

	var notif jsonRPCNotification
	if err := json.Unmarshal([]byte(line), &notif); err != nil {
		return
	}

	t.mu.Lock()
	handler, ok := t.notify[notif.Method]
	t.mu.Unlock()
	if ok {
		handler(notif.Params)
	}
}

func (t *Transport) detectCLIBinary() (string, error) {
	var binaryName string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			binaryName = "autohand-macos-arm64"
		} else {
			binaryName = "autohand-macos-x64"
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			binaryName = "autohand-linux-arm64"
		} else {
			binaryName = "autohand-linux-x64"
		}
	case "windows":
		binaryName = "autohand-windows-x64.exe"
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Try to find binary relative to this source file (../../cli/)
	if _, file, _, ok := runtime.Caller(0); ok {
		srcDir := filepath.Dir(file)
		packagePath := filepath.Join(srcDir, "..", "..", "cli", binaryName)
		if _, err := os.Stat(packagePath); err == nil {
			return packagePath, nil
		}
	}

	// Try current working directory
	if _, err := os.Stat(binaryName); err == nil {
		return binaryName, nil
	}

	// Try PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path, nil
	}

	// Fallback to generic "autohand" binary name in PATH
	if path, err := exec.LookPath("autohand"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("CLI binary not found: %s", binaryName)
}

func (t *Transport) log(format string, args ...interface{}) {
	if t.debug {
		fmt.Printf("[Transport] "+format+"\n", args...)
	}
}

// copySkillFiles copies custom SKILL.md files into ~/.autohand/skills/ before
// starting the CLI so the CLI can load them by name.
func (t *Transport) copySkillFiles(cfg *Config, cwd string) error {
	skillFiles := cfg.SkillRefs
	if len(skillFiles) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	autohandSkillsDir := filepath.Join(home, ".autohand", "skills")

	for _, ref := range skillFiles {
		path := GetSkillPath(ref)
		if path == "" {
			continue
		}

		srcPath := filepath.Join(cwd, path)
		if _, err := os.Stat(srcPath); err != nil {
			t.log("Skill file not found: %s", srcPath)
			continue
		}

		name := GetSkillName(ref)
		if name == "" {
			name = "custom-skill"
		}

		destDir := filepath.Join(autohandSkillsDir, name)
		destPath := filepath.Join(destDir, "SKILL.md")

		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.log("Failed to create skill dir: %s", err)
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.log("Failed to read skill file: %s", err)
			continue
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			t.log("Failed to write skill file: %s", err)
			continue
		}

		t.log("Copied skill file: %s -> %s", srcPath, destPath)
	}

	return nil
}
