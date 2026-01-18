// Package agent provides command execution framework tests for the SONAR agent
package agent

import (
    "context"
    "os"
    "testing"
    "time"
)

func TestCommandExecutor_NewCommandExecutor(t *testing.T) {
    executor := NewCommandExecutor()

    if executor == nil {
        t.Fatal("NewCommandExecutor returned nil")
    }

    if executor.maxOutputSize != 1024*1024 {
        t.Errorf("Expected max output size 1MB, got %d", executor.maxOutputSize)
    }

    if executor.maxConcurrent != 5 {
        t.Errorf("Expected max concurrent 5, got %d", executor.maxConcurrent)
    }

    if executor.rateLimit != 10 {
        t.Errorf("Expected rate limit 10, got %d", executor.rateLimit)
    }
}

func TestCommandExecutor_ValidateCommand(t *testing.T) {
    executor := NewCommandExecutor()

    tests := []struct {
        name      string
        command   string
        wantError bool
    }{
        {"Valid whitelisted command", "/bin/ls", false},
        {"Valid whitelisted command with args", "/bin/cat /etc/hosts", false},
        {"Dangerous command - rm -rf", "rm -rf /", true},
        {"Dangerous command - mkfs", "mkfs.ext4 /dev/sda", true},
        {"Dangerous command - dd", "dd if=/dev/zero of=/dev/sda", true},
        {"Dangerous command - shutdown", "shutdown -h now", true},
        {"Command injection - semicolon", "ls; rm -rf /", true},
        {"Command injection - pipe", "ls | rm -rf", true},
        {"Command injection - ampersand", "ls & rm -rf", true},
        {"Command injection - command substitution", "echo $(rm -rf)", true},
        {"Command injection - backtick", "echo `rm -rf`", true},
        {"Command injection - newline", "ls\nrm -rf", true},
        {"Not whitelisted command", "/usr/bin/vim", true},
        {"Command not in allowed path", "/tmp/script.sh", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := executor.ValidateCommand(tt.command)
            if (err != nil) != tt.wantError {
                t.Errorf("ValidateCommand() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCommandExecutor_ExecuteCommand(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    executor := NewCommandExecutor()
    ctx := context.Background()

    tests := []struct {
        name       string
        request    CommandRequest
        wantExit   int
        wantStdout bool
        wantError  bool
    }{
        {
            name: "Simple echo command",
            request: CommandRequest{
                Command: "/bin/echo",
                Args:    []string{"hello", "world"},
                Timeout: 10,
            },
            wantExit:   0,
            wantStdout: true,
            wantError:  false,
        },
        {
            name: "List current directory",
            request: CommandRequest{
                Command: "/bin/ls",
                Timeout: 10,
            },
            wantExit:   0,
            wantStdout: true,
            wantError:  false,
        },
        {
            name: "Command with working directory",
            request: CommandRequest{
                Command: "/bin/pwd",
                Workdir: "/tmp",
                Timeout: 10,
            },
            wantExit:   0,
            wantStdout: true,
            wantError:  false,
        },
        {
            name: "Command that fails",
            request: CommandRequest{
                Command: "/bin/ls",
                Args:    []string{"/nonexistent"},
                Timeout: 10,
            },
            wantExit:   2,
            wantStdout: false,
            wantError:  false,
        },
        {
            name: "Timeout",
            request: CommandRequest{
                Command: "/bin/sleep",
                Args:    []string{"5"},
                Timeout: 1,
            },
            wantExit:   -2,
            wantStdout: false,
            wantError:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            response, err := executor.ExecuteCommand(ctx, tt.request, "test")
            if err != nil {
                t.Errorf("ExecuteCommand() returned error: %v", err)
                return
            }

            if response.ExitCode != tt.wantExit {
                t.Errorf("ExecuteCommand() exitCode = %d, want %d", response.ExitCode, tt.wantExit)
            }

            if tt.wantStdout && len(response.Stdout) == 0 {
                t.Error("ExecuteCommand() expected stdout but got none")
            }

            if tt.wantError && response.Error == "" {
                t.Error("ExecuteCommand() expected error but got none")
            }
        })
    }
}

func TestCommandExecutor_RateLimit(t *testing.T) {
    executor := NewCommandExecutor()
    ctx := context.Background()

    // Create a fast command
    req := CommandRequest{
        Command: "/bin/echo",
        Args:    []string{"test"},
        Timeout: 5,
    }

    // Execute commands up to the rate limit (10 per minute)
    for i := 0; i < 10; i++ {
        response, err := executor.ExecuteCommand(ctx, req, "test")
        if err != nil {
            t.Fatalf("Command %d failed: %v", i, err)
        }
        if response.ExitCode != 0 {
            t.Fatalf("Command %d failed with exit code %d", i, response.ExitCode)
        }
    }

    // The next command should exceed the rate limit
    response, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Expected error for rate limit exceeded, got: %v", err)
    }

    if response.Error == "" {
        t.Error("Expected error message for rate limit exceeded")
    }

    if response.ExitCode != -1 {
        t.Errorf("Expected exit code -1 for rate limit, got %d", response.ExitCode)
    }
}

func TestCommandExecutor_ConcurrentLimit(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping concurrent test in short mode")
    }

    executor := NewCommandExecutor()
    ctx := context.Background()

    // Create long-running commands
    req := CommandRequest{
        Command: "/bin/sleep",
        Args:    []string{"2"},
        Timeout: 10,
    }

    // Execute 5 concurrent commands (max limit)
    results := make(chan *CommandResponse, 5)
    for i := 0; i < 5; i++ {
        go func() {
            response, err := executor.ExecuteCommand(ctx, req, "test")
            if err != nil {
                t.Errorf("Concurrent command failed: %v", err)
            }
            results <- response
        }()
    }

    // The 6th command should exceed the concurrent limit
    time.Sleep(100 * time.Millisecond) // Give the goroutines time to start
    response, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Expected error for concurrent limit exceeded, got: %v", err)
    }

    if response.Error == "" {
        t.Error("Expected error message for concurrent limit exceeded")
    }

    if response.ExitCode != -1 {
        t.Errorf("Expected exit code -1 for concurrent limit, got %d", response.ExitCode)
    }

    // Wait for the concurrent commands to complete
    for i := 0; i < 5; i++ {
        <-results
    }
}

func TestCommandExecutor_History(t *testing.T) {
    executor := NewCommandExecutor()
    ctx := context.Background()

    // Execute a command
    req := CommandRequest{
        Command: "/bin/echo",
        Args:    []string{"test"},
        Timeout: 10,
    }

    _, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Failed to execute command: %v", err)
    }

    // Check history
    history := executor.GetHistory(10)
    if len(history) == 0 {
        t.Fatal("Expected non-empty history")
    }

    if history[0].Command != "/bin/echo" {
        t.Errorf("Expected command /bin/echo, got %s", history[0].Command)
    }

    // Test limit
    history = executor.GetHistory(1)
    if len(history) != 1 {
        t.Errorf("Expected 1 history entry, got %d", len(history))
    }
}

func TestCommandExecutor_OutputLimit(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping output limit test in short mode")
    }

    executor := NewCommandExecutor()
    ctx := context.Background()

    // Create a large output
    tempFile, err := os.CreateTemp("", "large-output-*.txt")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tempFile.Name())

    // Write 2MB of data (more than the 1MB limit)
    largeData := make([]byte, 1024*1024*2)
    for i := range largeData {
        largeData[i] = 'A'
    }
    if _, err := tempFile.Write(largeData); err != nil {
        t.Fatalf("Failed to write to temp file: %v", err)
    }
    tempFile.Close()

    // Read the file
    req := CommandRequest{
        Command: "/bin/cat",
        Args:    []string{tempFile.Name()},
        Timeout: 10,
    }

    response, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Failed to execute command: %v", err)
    }

    // Check that output is limited
    if len(response.Stdout) > int(executor.maxOutputSize) {
        t.Errorf("Output size %d exceeds max %d", len(response.Stdout), executor.maxOutputSize)
    }
}

func TestCommandExecutor_EnvironmentVariables(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping environment variable test in short mode")
    }

    executor := NewCommandExecutor()
    ctx := context.Background()

    req := CommandRequest{
        Command: "/usr/bin/env",
        Timeout: 10,
        Env: map[string]string{
            "TEST_VAR":   "test_value",
            "ANOTHER_VAR": "another_value",
        },
    }

    response, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Failed to execute command: %v", err)
    }

    if response.ExitCode != 0 {
        t.Fatalf("Command failed with exit code %d", response.ExitCode)
    }

    // Check if environment variables are set
    if !contains(response.Stdout, "TEST_VAR=test_value") {
        t.Error("Environment variable TEST_VAR not set correctly")
    }

    if !contains(response.Stdout, "ANOTHER_VAR=another_value") {
        t.Error("Environment variable ANOTHER_VAR not set correctly")
    }
}

func TestCommandExecutor_EnvironmentVariableInjection(t *testing.T) {
    executor := NewCommandExecutor()
    ctx := context.Background()

    req := CommandRequest{
        Command: "/usr/bin/env",
        Timeout: 10,
        Env: map[string]string{
            "TEST_VAR": "test\nvalue", // Contains newline (injection attempt)
        },
    }

    response, err := executor.ExecuteCommand(ctx, req, "test")
    if err != nil {
        t.Fatalf("Failed to execute command: %v", err)
    }

    if response.Error == "" {
        t.Error("Expected error for environment variable injection")
    }

    if response.ExitCode != -1 {
        t.Errorf("Expected exit code -1, got %d", response.ExitCode)
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}
