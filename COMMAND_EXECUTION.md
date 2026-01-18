# Command Execution Framework

## Overview

The SONAR agent provides a secure, controlled framework for executing commands on monitored systems. This framework implements multiple security layers to prevent unauthorized or malicious command execution while providing flexibility for legitimate system administration tasks.

## Features

- **Command Validation**: Whitelist-based command filtering with configurable allowed commands
- **Security Controls**: Injection prevention, path validation, and dangerous command blocking
- **Rate Limiting**: Configurable limits on command execution frequency
- **Concurrent Execution**: Limits on simultaneous command executions
- **Output Limits**: Maximum output size to prevent memory exhaustion
- **Timeout Handling**: Configurable timeouts with automatic cancellation
- **Audit Logging**: Complete command history tracking for audit purposes
- **Environment Variables**: Support for custom environment variables with validation

## Configuration

### Environment Variables

Configure command execution behavior using the following environment variables:

#### `SONAR_AGENT_COMMAND_WHITELIST`

Defines which commands are allowed to be executed. Can be:

- `"allow_all"` - Allow all commands (development only, not recommended for production)
- JSON array: `["/usr/bin/curl", "/bin/ls", "/usr/bin/docker"]`
- Comma-separated: `/usr/bin/curl,/bin/ls,/usr/bin/docker`

**Default (production)**:
```json
[
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
  "/usr/bin/find"
]
```

#### `SONAR_AGENT_ALLOWED_PATHS`

Specifies which directories contain allowed commands. Commands using absolute paths must be within these directories.

- JSON array: `["/usr/bin", "/bin", "/usr/local/bin"]`
- Comma-separated: `/usr/bin,/bin,/usr/local/bin`

**Default**: `/usr/bin,/bin,/usr/local/bin,/usr/sbin,/sbin`

#### `SONAR_AGENT_COMMAND_MAX_OUTPUT`

Maximum size of command output (stdout + stderr) in bytes.

**Default**: `1048576` (1MB)

#### `SONAR_AGENT_COMMAND_RATE_LIMIT`

Maximum number of commands that can be executed per time window.

**Default**: `10`

#### `SONAR_AGENT_COMMAND_MAX_CONCURRENT`

Maximum number of commands that can execute simultaneously.

**Default**: `5`

### Hardcoded Security Rules

Certain commands are always blocked regardless of configuration:

- `rm -rf` - Force recursive delete
- `mkfs` - Filesystem creation
- `dd` - Disk copying/formatting
- `fdisk` - Disk partitioning
- `parted` - Partition management
- `shutdown` - System shutdown
- `reboot` - System reboot
- `halt` - System halt
- `poweroff` - System power off
- `:(){:|:&};:` - Fork bomb
- Command injection patterns: `;`, `|`, `&`, `$(`, `` ` ``, newline, carriage return, tab

## API Endpoints

### Execute Command

**Endpoint**: `POST /api/sonar/agents/{agentId}/commands/execute`

**Authentication**: Required

**Request Body**:
```json
{
  "command": "/bin/echo",
  "args": ["hello", "world"],
  "timeout": 30,
  "workdir": "/tmp",
  "env": {
    "MY_VAR": "value"
  }
}
```

**Parameters**:
- `command` (string, required): Command to execute
- `args` (array, optional): Command arguments
- `timeout` (integer, optional): Timeout in seconds (default: 30, max: 300)
- `workdir` (string, optional): Working directory for command execution
- `env` (object, optional): Environment variables to set

**Response**:
```json
{
  "exitCode": 0,
  "stdout": "hello world\n",
  "stderr": "",
  "error": "",
  "duration": 5
}
```

**Exit Codes**:
- `0`: Success
- `> 0`: Command failed with specific exit code
- `-1`: Execution error (validation failed, not found, etc.)
- `-2`: Timeout
- `127`: Command not found

**Error Responses**:
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `404 Not Found`: Agent not found
- `503 Service Unavailable`: Agent is offline
- `504 Gateway Timeout`: Command execution timed out
- `500 Internal Server Error`: Server error

### Get Command History

**Endpoint**: `GET /api/sonar/agents/{agentId}/commands/history`

**Authentication**: Required

**Query Parameters**:
- `limit` (integer, optional): Maximum number of history entries (default: 10, max: 100)

**Response**:
```json
{
  "history": [
    {
      "command": "/bin/echo",
      "args": ["hello"],
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": 5,
      "exitCode": 0,
      "workdir": "/tmp"
    }
  ]
}
```

## Usage Examples

### Basic Command Execution

```bash
# Execute a simple echo command
curl -X POST http://localhost:8090/api/sonar/agents/{agentId}/commands/execute \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "/bin/echo",
    "args": ["hello", "world"]
  }'
```

### Command with Working Directory

```bash
# List files in a specific directory
curl -X POST http://localhost:8090/api/sonar/agents/{agentId}/commands/execute \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "/bin/ls",
    "workdir": "/var/log"
  }'
```

### Command with Environment Variables

```bash
# Run a command with custom environment variables
curl -X POST http://localhost:8090/api/sonar/agents/{agentId}/commands/execute \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "/usr/bin/env",
    "env": {
      "MY_APP": "production",
      "DEBUG": "false"
    }
  }'
```

### Get Command History

```bash
# Get the last 20 commands executed
curl -X GET "http://localhost:8090/api/sonar/agents/{agentId}/commands/history?limit=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Security Best Practices

### 1. Use Strict Whitelisting

Only allow commands that are necessary for your use case:

```bash
export SONAR_AGENT_COMMAND_WHITELIST='["/bin/ls","/bin/cat","/usr/bin/docker"]'
```

### 2. Restrict Command Paths

Limit allowed command directories:

```bash
export SONAR_AGENT_ALLOWED_PATHS='["/usr/bin","/bin"]'
```

### 3. Set Conservative Limits

Prevent abuse with conservative limits:

```bash
export SONAR_AGENT_COMMAND_RATE_LIMIT=5
export SONAR_AGENT_COMMAND_MAX_CONCURRENT=3
export SONAR_AGENT_COMMAND_MAX_OUTPUT=524288  # 512KB
```

### 4. Monitor Command History

Regularly review command history for suspicious activity:

```bash
curl -X GET "http://localhost:8090/api/sonar/agents/{agentId}/commands/history?limit=100" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 5. Use Timeouts

Always set appropriate timeouts to prevent hanging commands:

```json
{
  "command": "/usr/bin/docker",
  "args": ["ps"],
  "timeout": 10
}
```

### 6. Validate Before Execution

Always validate commands on the hub side before sending to the agent:

```javascript
// Example: Validate command before execution
const allowedCommands = ['/bin/ls', '/bin/cat'];
if (!allowedCommands.includes(req.command)) {
  return res.status(400).json({error: 'Command not allowed'});
}
```

## Common Use Cases

### System Information Gathering

```bash
# Get system hostname
POST /api/sonar/agents/{agentId}/commands/execute
{"command": "/bin/hostname"}

# Get system uptime
POST /api/sonar/agents/{agentId}/commands/execute
{"command": "/usr/bin/uptime"}

# Get current user
POST /api/sonar/agents/{agentId}/commands/execute
{"command": "/bin/whoami"}
```

### Docker Management

```bash
# List running containers
POST /api/sonar/agents/{agentId}/commands/execute
{"command": "/usr/bin/docker", "args": ["ps"]}

# View container logs
POST /api/sonar/agents/{agentId}/commands/execute
{
  "command": "/usr/bin/docker",
  "args": ["logs", "container_id"],
  "timeout": 30
}
```

### Service Management

```bash
# Check service status
POST /api/sonar/agents/{agentId}/commands/execute
{
  "command": "/usr/bin/systemctl",
  "args": ["status", "nginx"]
}

# Restart a service (if whitelisted)
POST /api/sonar/agents/{agentId}/commands/execute
{
  "command": "/usr/bin/systemctl",
  "args": ["restart", "nginx"],
  "timeout": 60
}
```

### File Operations

```bash
# Read a file
POST /api/sonar/agents/{agentId}/commands/execute
{
  "command": "/bin/cat",
  "args": ["/etc/hosts"]
}

# Find files
POST /api/sonar/agents/{agentId}/commands/execute
{
  "command": "/usr/bin/find",
  "args": ["/var/log", "-name", "*.log"],
  "timeout": 30
}
```

## Troubleshooting

### Command Not Allowed

**Error**: `command validation failed: command not in whitelist: /usr/bin/vim`

**Solution**: Add the command to the whitelist:
```bash
export SONAR_AGENT_COMMAND_WHITELIST='["/usr/bin/vim"]'
```

### Command Contains Injection Pattern

**Error**: `command validation failed: command contains injection pattern: ;`

**Solution**: Remove shell metacharacters from the command string. Pass arguments separately:
```json
{
  "command": "/bin/ls",
  "args": ["-la", "/tmp"]
}
```

### Rate Limit Exceeded

**Error**: `rate limit exceeded: 10 commands per 1m0s`

**Solution**: Wait for the rate limit window to expire or increase the limit:
```bash
export SONAR_AGENT_COMMAND_RATE_LIMIT=20
```

### Concurrent Limit Reached

**Error**: `maximum concurrent command execution limit reached`

**Solution**: Wait for existing commands to complete or increase the limit:
```bash
export SONAR_AGENT_COMMAND_MAX_CONCURRENT=10
```

### Command Timed Out

**Error**: `command execution timed out`

**Solution**: Increase the timeout or optimize the command:
```json
{
  "command": "/usr/bin/docker",
  "args": ["ps"],
  "timeout": 60
}
```

## Advanced Configuration

### Custom Whitelist by Environment

Set different whitelists for different environments:

```bash
# Development
export SONAR_AGENT_COMMAND_WHITELIST="allow_all"

# Production
export SONAR_AGENT_COMMAND_WHITELIST='["/bin/ls","/bin/cat","/usr/bin/docker"]'
```

### Network-Specific Configuration

Configure different limits based on network:

```bash
# Trusted network
export SONAR_AGENT_COMMAND_RATE_LIMIT=20
export SONAR_AGENT_COMMAND_MAX_CONCURRENT=10

# Untrusted network
export SONAR_AGENT_COMMAND_RATE_LIMIT=5
export SONAR_AGENT_COMMAND_MAX_CONCURRENT=2
```

## Migration Notes

### From Direct SSH Execution

When migrating from direct SSH execution:

1. **Review Commands**: Audit existing commands for security compliance
2. **Update Whitelist**: Add necessary commands to the whitelist
3. **Adjust Limits**: Set appropriate rate and concurrent limits
4. **Update Code**: Change from SSH libraries to SONAR API

### Example Migration

**Before (SSH)**:
```python
import paramiko
client = paramiko.SSHClient()
client.connect('hostname', username='user')
stdin, stdout, stderr = client.exec_command('ls -la /tmp')
```

**After (SONAR API)**:
```python
import requests

response = requests.post(
    'http://localhost:8090/api/sonar/agents/{agentId}/commands/execute',
    headers={'Authorization': 'Bearer TOKEN'},
    json={
        'command': '/bin/ls',
        'args': ['-la', '/tmp']
    }
)
result = response.json()
print(result['stdout'])
```

## Performance Considerations

- **Memory Usage**: Each command execution captures up to `COMMAND_MAX_OUTPUT` bytes
- **Concurrent Commands**: Each concurrent command uses memory for output buffering
- **History Storage**: Last 100 commands kept in memory per agent
- **Rate Limiting**: Helps prevent resource exhaustion from excessive commands

## Future Enhancements

Planned features for future versions:

- [ ] Command scheduling (execute at specific times)
- [ ] Command templates with parameterization
- [ ] Enhanced logging with structured output
- [ ] Command output streaming for long-running commands
- [ ] Command chaining (dependant commands)
- [ ] SSH-based command execution as fallback
- [ ] Command result caching
