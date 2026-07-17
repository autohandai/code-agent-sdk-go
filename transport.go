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
		nextID:    1,
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

	args := buildCLIArgs(cfg)

	t.log("Starting CLI: %s", cliPath)
	t.log("Args: %v", args)

	env := buildCLIEnv(cfg, os.Environ())

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

func buildCLIArgs(cfg *Config) []string {
	args := []string{"--mode", "rpc"}
	flag := func(ok bool, name string) {
		if ok {
			args = append(args, name)
		}
	}
	value := func(name, v string) {
		if v != "" {
			args = append(args, name, v)
		}
	}
	flag(cfg.Bare, "--bare")
	flag(cfg.Unrestricted, "--unrestricted")
	flag(cfg.AutoMode, "--auto-mode")
	flag(cfg.AutoSkill, "--auto-skill")
	flag(cfg.AutoCommit, "-c")
	if cfg.IdleLogout != nil && !*cfg.IdleLogout {
		args = append(args, "--no-idle-logout")
	}
	flag(cfg.ContextCompact, "--context-compact")
	flag(cfg.NoContextCompact, "--no-context-compact")
	flag(cfg.PersistSession, "--persist-session")
	value("--session-id", cfg.SessionID)
	flag(cfg.Resume, "--resume")
	flag(cfg.Continue, "--continue")
	flag(cfg.Fork, "--fork")
	value("--session-path", cfg.SessionPath)
	if cfg.AutoSaveInterval > 0 {
		value("--auto-save-interval", fmt.Sprint(cfg.AutoSaveInterval))
	}
	flag(cfg.NoAgentsMd, "--no-agents-md")
	flag(cfg.AgentsMdEnable, "--agents-md")
	flag(cfg.AgentsMdCreate, "--agents-md-create")
	value("--agents-md-path", cfg.AgentsMdPath)
	flag(cfg.AgentsMdAutoUpdate, "--agents-md-auto-update")
	if cfg.MaxTokens > 0 {
		value("--max-tokens", fmt.Sprint(cfg.MaxTokens))
	}
	if cfg.CompressionThreshold > 0 {
		value("--compression-threshold", fmt.Sprint(cfg.CompressionThreshold))
	}
	if cfg.SummarizationThreshold > 0 {
		value("--summarization-threshold", fmt.Sprint(cfg.SummarizationThreshold))
	}
	if len(cfg.Skills) > 0 {
		names := make([]string, len(cfg.Skills))
		for i, s := range cfg.Skills {
			names[i] = s.Name
		}
		value("--skills", strings.Join(names, ","))
	}
	if len(cfg.SkillSources) > 0 {
		value("--skill-sources", strings.Join(cfg.SkillSources, ","))
	}
	flag(cfg.InstallMissingSkills, "--install-missing-skills")
	if cfg.MaxIterations > 0 {
		value("--max-iterations", fmt.Sprint(cfg.MaxIterations))
	}
	if cfg.MaxRuntime > 0 {
		value("--max-runtime", fmt.Sprint(cfg.MaxRuntime))
	}
	if cfg.MaxCost > 0 {
		value("--max-cost", fmt.Sprint(cfg.MaxCost))
	}
	value("--display-language", cfg.DisplayLanguage)
	value("--sys-prompt", cfg.SysPrompt)
	value("--system-prompt-file", cfg.SystemPromptFile)
	value("--append-sys-prompt", cfg.AppendSysPrompt)
	value("--append-system-prompt-file", cfg.AppendSystemPromptFile)
	value("--mcp-config", cfg.MCPConfig)
	value("--agents", cfg.Agents)
	value("--plugin-dir", cfg.PluginDir)
	value("--model", cfg.Model)
	if cfg.Temperature > 0 {
		value("--temperature", fmt.Sprint(cfg.Temperature))
	}
	value("--yolo", cfg.Yolo)
	if cfg.YoloTimeout > 0 {
		value("--yolo-timeout", fmt.Sprint(cfg.YoloTimeout))
	}
	for _, dir := range append(append([]string{}, cfg.AddDir...), cfg.AdditionalDirectories...) {
		value("--add-dir", dir)
	}
	return append(args, cfg.ExtraArgs...)
}

func buildCLIEnv(cfg *Config, base []string) []string {
	env := append(append([]string{}, base...), "AUTOHAND_STREAM_TOOL_OUTPUT=1")
	if cfg.Provider == ProviderAutohandAI {
		plan := cfg.AutohandAIPlan
		if plan == "" {
			plan = "cloud"
		}
		env = append(env, "AUTOHAND_AI_PLAN="+plan)
		if cfg.APIKey != "" {
			env = append(env, "AUTOHAND_AI_API_KEY="+cfg.APIKey)
		}
		if cfg.BaseURL != "" {
			env = append(env, "AUTOHAND_AI_BASE_URL="+cfg.BaseURL)
		}
	}
	for _, values := range []map[string]string{cfg.Env, cfg.EnvVars} {
		for k, v := range values {
			env = append(env, k+"="+v)
		}
	}
	return env
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
