// Package agent provides command execution framework for the SONAR agent
package agent

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

// CommandRequest represents a command execution request
type CommandRequest struct {
    Command string            `json:"command"`
    Args    []string          `json:"args,omitempty"`
    Timeout int               `json:"timeout"` // timeout in seconds, default 30
    Workdir string            `json:"workdir,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
}

// CommandResponse represents the result of a command execution
type CommandResponse struct {
    ExitCode int    `json:"exitCode"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
    Error    string `json:"error,omitempty"`
    Duration int64  `json:"duration"` // in milliseconds
}

// CommandHistoryEntry stores information about a executed command
type CommandHistoryEntry struct {
    Command   string        `json:"command"`
    Args      []string      `json:"args"`
    Timestamp time.Time     `json:"timestamp"`
    Duration  time.Duration `json:"duration"`
    ExitCode  int           `json:"exitCode"`
    Workdir   string        `json:"workdir"`
}

// CommandExecutor manages command execution with security controls
type CommandExecutor struct {
    whitelist           []string
    blacklist           []string
    allowedPaths        []string
    history             []CommandHistoryEntry
    historyMutex        sync.Mutex
    rateLimitTracker    map[string][]time.Time
    rateLimitMutex      sync.Mutex
    concurrentSemaphore chan struct{}
    maxOutputSize       int64
    maxConcurrent       int
    rateLimit           int
    rateLimitWindow     time.Duration
}

// NewCommandExecutor creates a new command executor with default settings
func NewCommandExecutor() *CommandExecutor {
    executor := &CommandExecutor{
        history:           make([]CommandHistoryEntry, 0, 100),
        rateLimitTracker:  make(map[string][]time.Time),
        maxOutputSize:     1024 * 1024, // 1MB default
        maxConcurrent:     5,
        rateLimit:         10,
        rateLimitWindow:   time.Minute,
        concurrentSemaphore: make(chan struct{}, 5),
    }

    // Load configuration from environment
    executor.loadConfig()

    // Fill semaphore with available slots
    for i := 0; i < executor.maxConcurrent; i++ {
        executor.concurrentSemaphore <- struct{}{}
    }

    return executor
}

// loadConfig loads configuration from environment variables
func (ce *CommandExecutor) loadConfig() {
    // Load whitelist
    if whitelist, exists := GetEnv("COMMAND_WHITELIST"); exists {
        if whitelist == "allow_all" {
            ce.whitelist = nil // Allow all commands
        } else if strings.HasPrefix(whitelist, "[") {
            // JSON array format
            var commands []string
            if err := json.Unmarshal([]byte(whitelist), &commands); err == nil {
                ce.whitelist = commands
            }
        } else {
            // Comma-separated format
            ce.whitelist = strings.Split(whitelist, ",")
            for i := range ce.whitelist {
                ce.whitelist[i] = strings.TrimSpace(ce.whitelist[i])
            }
        }
    } else {
        // Default whitelist for production
        ce.whitelist = []string{
            "/usr/bin/curl",
            "/usr/bin/wget",
            "/bin/ps",
            "/usr/bin/systemctl",
            "/bin/hostname",
            "/usr/bin/uptime",
            "/bin/date",
            "/bin/whoami",
            "/usr/bin/docker",
            "/usr/local/bin/docker",
            "/bin/cat",
            "/usr/bin/head",
            "/usr/bin/tail",
            "/bin/ls",
            "/usr/bin/find",
            "/bin/echo",
            "/bin/pwd",
            "/bin/sleep",
            "/usr/bin/env",
        }
    }

    // Load allowed paths
    if paths, exists := GetEnv("ALLOWED_PATHS"); exists {
        if strings.HasPrefix(paths, "[") {
            // JSON array format
            var pathList []string
            if err := json.Unmarshal([]byte(paths), &pathList); err == nil {
                ce.allowedPaths = pathList
            }
        } else {
            // Comma-separated format
            ce.allowedPaths = strings.Split(paths, ",")
            for i := range ce.allowedPaths {
                ce.allowedPaths[i] = strings.TrimSpace(ce.allowedPaths[i])
            }
        }
    } else {
        // Default allowed paths
        ce.allowedPaths = []string{
            "/usr/bin",
            "/bin",
            "/usr/local/bin",
            "/usr/sbin",
            "/sbin",
        }
    }

    // Load dangerous commands blacklist (always blocked)
    ce.blacklist = []string{
        "rm -rf",
        "mkfs",
        "dd ",
        "fdisk",
        "parted",
        "shutdown",
        "reboot",
        "halt",
        "poweroff",
        ":(){:|:&};:", // fork bomb
    }

    // Load max output size
    if maxSize, exists := GetEnv("COMMAND_MAX_OUTPUT"); exists {
        var size int64
        if _, err := fmt.Sscanf(maxSize, "%d", &size); err == nil {
            ce.maxOutputSize = size
        }
    }

    // Load rate limit
    if rateLimit, exists := GetEnv("COMMAND_RATE_LIMIT"); exists {
        var limit int
        if _, err := fmt.Sscanf(rateLimit, "%d", &limit); err == nil && limit > 0 {
            ce.rateLimit = limit
        }
    }

    // Load max concurrent
    if maxConcurrent, exists := GetEnv("COMMAND_MAX_CONCURRENT"); exists {
        var max int
        if _, err := fmt.Sscanf(maxConcurrent, "%d", &max); err == nil && max > 0 {
            ce.maxConcurrent = max
        }
    }
}

// ValidateCommand checks if a command is allowed to be executed
func (ce *CommandExecutor) ValidateCommand(command string) error {
    // Check against dangerous commands
    for _, dangerous := range ce.blacklist {
        if strings.Contains(command, dangerous) {
            return fmt.Errorf("command contains dangerous pattern: %s", dangerous)
        }
    }

    // Check command injection attempts
    injectionPatterns := []string{
        ";", "|", "&", "$(", "`", "\n", "\r", "\t",
    }
    for _, pattern := range injectionPatterns {
        if strings.Contains(command, pattern) {
            return fmt.Errorf("command contains injection pattern: %s", pattern)
        }
    }

    // Check path if using absolute path
    if filepath.IsAbs(command) {
        commandDir := filepath.Dir(command)
        allowed := false
        for _, path := range ce.allowedPaths {
            if strings.HasPrefix(commandDir, path) {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("command path not in allowed paths: %s", commandDir)
        }
    }

    // Check whitelist (if not "allow_all")
    if ce.whitelist != nil {
        allowed := false
        for _, allowedCmd := range ce.whitelist {
            if command == allowedCmd || strings.HasPrefix(command, allowedCmd+" ") {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("command not in whitelist: %s", command)
        }
    }

    return nil
}

// checkRateLimit checks if the command execution should be allowed based on rate limiting
func (ce *CommandExecutor) checkRateLimit(key string) error {
    ce.rateLimitMutex.Lock()
    defer ce.rateLimitMutex.Unlock()

    now := time.Now()
    timestamps := ce.rateLimitTracker[key]

    // Remove timestamps outside the window
    var validTimestamps []time.Time
    for _, ts := range timestamps {
        if now.Sub(ts) < ce.rateLimitWindow {
            validTimestamps = append(validTimestamps, ts)
        }
    }

    // Check if rate limit exceeded
    if len(validTimestamps) >= ce.rateLimit {
        return fmt.Errorf("rate limit exceeded: %d commands per %v", ce.rateLimit, ce.rateLimitWindow)
    }

    // Add current timestamp
    validTimestamps = append(validTimestamps, now)
    ce.rateLimitTracker[key] = validTimestamps

    return nil
}

// addToHistory adds a command to the execution history
func (ce *CommandExecutor) addToHistory(entry CommandHistoryEntry) {
    ce.historyMutex.Lock()
    defer ce.historyMutex.Unlock()

    ce.history = append(ce.history, entry)

    // Keep only last 100 entries
    if len(ce.history) > 100 {
        ce.history = ce.history[len(ce.history)-100:]
    }
}

// GetHistory returns the command history
func (ce *CommandExecutor) GetHistory(limit int) []CommandHistoryEntry {
    ce.historyMutex.Lock()
    defer ce.historyMutex.Unlock()

    if limit <= 0 || limit > len(ce.history) {
        limit = len(ce.history)
    }

    // Return the most recent entries
    start := len(ce.history) - limit
    return ce.history[start:]
}

// limitedWriter wraps a writer and limits the amount of data written
type limitedWriter struct {
    writer    *bytes.Buffer
    maxSize   int64
    bytesLeft int64
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
    if lw.bytesLeft <= 0 {
        return len(p), nil
    }

    if int64(len(p)) > lw.bytesLeft {
        n = int(lw.bytesLeft)
        lw.writer.Write(p[:n])
        lw.bytesLeft = 0
        return n, nil
    }

    lw.writer.Write(p)
    lw.bytesLeft -= int64(len(p))
    return len(p), nil
}

// ExecuteCommand executes a system command with the given parameters
func (ce *CommandExecutor) ExecuteCommand(ctx context.Context, req CommandRequest, clientKey string) (*CommandResponse, error) {
    startTime := time.Now()

    // Set default timeout
    if req.Timeout <= 0 {
        req.Timeout = 30 // default 30 seconds
    }
    if req.Timeout > 300 {
        req.Timeout = 300 // max 5 minutes
    }

    // Validate command
    if err := ce.ValidateCommand(req.Command); err != nil {
        return &CommandResponse{
            Error:    fmt.Sprintf("command validation failed: %s", err.Error()),
            ExitCode: -1,
        }, nil
    }

    // Check rate limit
    if err := ce.checkRateLimit(clientKey); err != nil {
        return &CommandResponse{
            Error:    err.Error(),
            ExitCode: -1,
        }, nil
    }

    // Check concurrent execution limit
    select {
    case <-ce.concurrentSemaphore:
        defer func() { ce.concurrentSemaphore <- struct{}{} }()
    default:
        return &CommandResponse{
            Error:    "maximum concurrent command execution limit reached",
            ExitCode: -1,
        }, nil
    }

    // Create command context with timeout
    cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
    defer cancel()

    // Create command
    cmd := exec.CommandContext(cmdCtx, req.Command, req.Args...)

    // Set working directory
    if req.Workdir != "" {
        cmd.Dir = req.Workdir
    }

    // Set environment variables
    if req.Env != nil {
        env := os.Environ()
        for k, v := range req.Env {
            // Validate environment variable to prevent injection
            if strings.ContainsAny(k, "\n\r") || strings.ContainsAny(v, "\n\r") {
                return &CommandResponse{
                    Error:    "environment variable contains invalid characters",
                    ExitCode: -1,
                }, nil
            }
            env = append(env, fmt.Sprintf("%s=%s", k, v))
        }
        cmd.Env = env
    }

    // Capture stdout and stderr with size limits
    var stdoutBuf, stderrBuf bytes.Buffer
    stdoutWriter := &limitedWriter{
        writer:    &stdoutBuf,
        maxSize:   ce.maxOutputSize,
        bytesLeft: ce.maxOutputSize,
    }
    stderrWriter := &limitedWriter{
        writer:    &stderrBuf,
        maxSize:   ce.maxOutputSize,
        bytesLeft: ce.maxOutputSize,
    }

    cmd.Stdout = stdoutWriter
    cmd.Stderr = stderrWriter

    // Execute command
    err := cmd.Run()
    duration := time.Since(startTime)

    // Build response
    response := &CommandResponse{
        Stdout:   stdoutBuf.String(),
        Stderr:   stderrBuf.String(),
        Duration: duration.Milliseconds(),
    }

    // Determine exit code
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            response.ExitCode = exitErr.ExitCode()
            response.Error = err.Error()

            // Check for timeout
            if cmdCtx.Err() == context.DeadlineExceeded {
                response.Error = "command execution timed out"
                response.ExitCode = -2
            }
        } else {
            // Command not found or other error
            if os.IsNotExist(err) {
                response.Error = "command not found"
                response.ExitCode = 127
            } else {
                response.Error = err.Error()
                response.ExitCode = -1
            }
        }
    }

    // Log command execution
    slog.Info("Command executed",
        "command", req.Command,
        "args", req.Args,
        "exitCode", response.ExitCode,
        "duration", duration,
        "workdir", req.Workdir,
    )

    // Add to history
    ce.addToHistory(CommandHistoryEntry{
        Command:   req.Command,
        Args:      req.Args,
        Timestamp: startTime,
        Duration:  duration,
        ExitCode:  response.ExitCode,
        Workdir:   req.Workdir,
    })

    return response, nil
}

// ExecuteCommand executes a system command with the given parameters
func (a *Agent) ExecuteCommand(ctx context.Context, req CommandRequest) (*CommandResponse, error) {
    slog.Info("Command execution requested", "command", req.Command, "args", req.Args)

    if a.commandExecutor == nil {
        a.commandExecutor = NewCommandExecutor()
    }

    // Use a simple key for rate limiting (could be improved with authentication)
    clientKey := "default"

    return a.commandExecutor.ExecuteCommand(ctx, req, clientKey)
}

// ValidateCommand checks if a command is allowed to be executed
func (a *Agent) ValidateCommand(command string) error {
    if a.commandExecutor == nil {
        a.commandExecutor = NewCommandExecutor()
    }
    return a.commandExecutor.ValidateCommand(command)
}

// GetCommandHistory returns the history of executed commands
func (a *Agent) GetCommandHistory(limit int) ([]CommandHistoryEntry, error) {
    if a.commandExecutor == nil {
        return []CommandHistoryEntry{}, nil
    }
    return a.commandExecutor.GetHistory(limit), nil
}
