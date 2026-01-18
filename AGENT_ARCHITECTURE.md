# SONAR Agent Architecture

This document provides technical details about the SONAR agent architecture, design decisions, and extension points.

## Overview

The SONAR agent is a lightweight monitoring daemon written in Go that collects system metrics and communicates with the SONAR hub. It supports multiple connection methods (WebSocket, SSH) and can monitor various system resources.

## Architecture Components

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                        SONAR Agent                          │
├─────────────────────────────────────────────────────────────┤
│  Connection Manager                                         │
│  ├─ WebSocket Client                                        │
│  └─ SSH Server                                              │
├─────────────────────────────────────────────────────────────┤
│  Data Collection                                            │
│  ├─ System Metrics (CPU, Memory, Disk, Network)            │
│  ├─ Docker/Container Monitoring                            │
│  ├─ GPU Monitoring (NVIDIA, AMD, Intel)                    │
│  ├─ SMART Monitoring                                        │
│  ├─ Systemd Service Monitoring                             │
│  ├─ Temperature Sensors                                     │
│  └─ Battery Monitoring                                      │
├─────────────────────────────────────────────────────────────┤
│  Extension Framework (Phase 0.1)                            │
│  ├─ Command Execution (stub)                               │
│  ├─ Service Management (stub)                              │
│  └─ File Operations (stub)                                 │
├─────────────────────────────────────────────────────────────┤
│  Cache & Storage                                            │
│  ├─ System Data Cache                                      │
│  ├─ Delta Tracking                                          │
│  └─ Persistent Storage                                      │
└─────────────────────────────────────────────────────────────┘
```

## Connection Management

### WebSocket Connection

The agent maintains a WebSocket connection to the hub for real-time communication:

- **Protocol**: Binary WebSocket with CBOR encoding
- **Reconnection**: Automatic reconnection with exponential backoff
- **Authentication**: Token-based authentication with signature verification
- **Compression**: CBOR binary format for efficient data transfer

### SSH Fallback

If WebSocket connection fails, the agent can accept SSH connections from the hub:

- **Key-based authentication**: ED25519 public key verification
- **Port forwarding**: Supports SSH tunneling
- **Fallback mode**: Used when WebSocket is unavailable

## Data Collection

### System Metrics

The agent collects the following system metrics:

#### CPU
- **Usage**: Per-core and total CPU usage
- **Load Average**: 1, 5, and 15-minute load averages
- **Implementation**: Uses `gopsutil/cpu`

#### Memory
- **RAM**: Total, used, available, cached
- **Swap**: Total, used, free
- **ZFS ARC**: Automatic detection and monitoring
- **Formula**: Configurable memory calculation (available, used, etc.)

#### Disk
- **Usage**: Per-partition disk usage
- **I/O**: Read/write bytes and operations per second
- **Multiple filesystems**: Supports monitoring multiple mount points
- **Cache**: Configurable cache duration to prevent waking sleeping disks

#### Network
- **Bandwidth**: Upload/download bytes per second
- **Per-interface**: Individual network interface monitoring
- **Delta tracking**: Efficient calculation of rates

### Container Monitoring

The agent monitors Docker/Podman containers:

- **Metrics**: CPU, memory, network usage per container
- **Status**: Running, stopped, paused states
- **Logs**: Container log retrieval (via hub request)
- **Info**: Container details and configuration

### GPU Monitoring

#### NVIDIA GPUs
- **NVML library**: Direct NVIDIA Management Library integration
- **Metrics**: Utilization, memory, temperature, power draw
- **Multi-GPU**: Supports multiple GPUs

#### AMD GPUs
- **ROCm**: AMD ROCm platform support
- **Metrics**: Similar to NVIDIA

#### Intel GPUs
- **Intel GPU tools**: Integration with Intel GPU monitoring

### SMART Monitoring

The agent can retrieve SMART disk health data:

- **Health status**: Overall disk health
- **Attributes**: Temperature, reallocated sectors, etc.
- **Refresh**: On-demand refresh of SMART data
- **Multiple disks**: Monitors all available disks

### Systemd Services (Linux)

On Linux systems, the agent can monitor systemd services:

- **Service status**: Active, inactive, failed states
- **Service details**: Memory usage, CPU usage, uptime
- **Info retrieval**: Detailed service information on demand

### Temperature Sensors

The agent monitors system temperature sensors:

- **CPU temperature**: Per-core and package temperatures
- **System sensors**: All available temperature sensors
- **Platform-specific**: Different implementations for Linux/Windows

### Battery Monitoring

For laptops and portable devices:

- **Charge level**: Current battery percentage
- **State**: Charging, discharging, full
- **Power draw**: Current power consumption

## Data Caching

### Cache Strategy

The agent implements intelligent caching to reduce system load:

```go
type systemDataCache struct {
    mu    sync.RWMutex
    cache map[uint16]*cacheEntry  // Keyed by cache time in milliseconds
}
```

- **Multi-level cache**: Different cache durations for different use cases
- **TTL**: Each cache entry has a time-to-live
- **Automatic cleanup**: Expired entries are automatically removed

### Delta Tracking

For metrics that require rate calculation (network, disk I/O):

```go
type DeltaTracker[K comparable, V constraints.Integer] struct {
    mu      sync.RWMutex
    current map[K]V
    prev    map[K]V
}
```

- **Previous values**: Stores previous measurements
- **Rate calculation**: Efficiently calculates deltas
- **Per-interface**: Separate tracking for each network interface

## Extension Framework

### Phase 0.1 Implementation

The agent includes stub implementations for future functionality:

#### Command Execution (`agent/cmd.go`)
- **Framework**: Command execution structure
- **Validation**: Command whitelist/blacklist support (stub)
- **Timeout**: Configurable command timeouts
- **Status**: Not yet implemented

#### Service Management (`agent/services.go`)
- **Operations**: Start, stop, restart, enable, disable
- **Status**: Service status retrieval
- **Validation**: Permission and service validation (stub)
- **Status**: Not yet implemented

#### File Operations (`agent/files.go`)
- **Operations**: Read, write, list, delete
- **Validation**: Path validation and security checks (stub)
- **Permissions**: File permission management
- **Status**: Not yet implemented

### WebSocket Actions

New WebSocket action types have been added:

```go
const (
    // Existing actions...
    GetData
    CheckFingerprint
    GetContainerLogs
    GetContainerInfo
    GetSmartData
    GetSystemdInfo
    
    // Phase 0.1 stub actions
    ExecuteCommand
    ListServices
    GetServiceStatus
    PerformServiceAction
    ListFiles
    ReadFile
    WriteFile
    DeleteFile
)
```

## Security Considerations

### Authentication

- **Token-based**: Each agent uses a unique authentication token
- **Signature verification**: Public key signatures for fingerprint verification
- **SSH keys**: ED25519 key pairs for SSH connections

### Data Protection

- **Read-only Docker socket**: Docker socket is mounted read-only
- **Restricted paths**: File operations will validate paths against restricted list
- **Command whitelist**: Command execution will support whitelisting

### Network Security

- **WebSocket over TLS**: Supports WSS for encrypted communication
- **SSH tunneling**: Alternative secure connection method
- **No inbound connections**: Agent connects to hub, not vice versa (for WebSocket mode)

## Performance Optimization

### Resource Usage

The agent is designed to be lightweight:

- **Minimal memory**: Typically <50MB RAM usage
- **Low CPU**: <1% CPU usage during normal operation
- **Efficient caching**: Reduces unnecessary system calls

### Optimizations

- **GOGC tuning**: Garbage collector tuning for reduced memory
- **Lazy initialization**: Components initialized on first use
- **Delta tracking**: Efficient rate calculations
- **Binary protocol**: CBOR encoding for compact data transfer

## Platform Support

### Linux
- Full feature support
- Systemd integration
- GPU monitoring (NVIDIA, AMD, Intel)
- SMART monitoring
- Temperature sensors

### Windows
- Core monitoring features
- Limited GPU support
- Windows service management (future)
- No systemd support (obviously)

### macOS
- Core monitoring features
- Limited GPU support
- Battery monitoring
- No systemd support

### FreeBSD/Other UNIX
- Core monitoring features
- Limited advanced features

## Building the Agent

### Standard Build

```bash
make build-agent
```

### Platform-Specific

```bash
# Linux
GOOS=linux GOARCH=amd64 make build-agent

# Windows
GOOS=windows GOARCH=amd64 make build-agent

# macOS
GOOS=darwin GOARCH=arm64 make build-agent
```

### Docker Images

Multiple Docker image variants:

- **scratch**: Minimal image based on scratch (~10MB)
- **alpine**: Alpine-based image with shell (~15MB)
- **nvidia**: NVIDIA GPU support (~100MB)
- **intel**: Intel GPU support (~50MB)

## Extending the Agent

### Adding New Metrics

1. Implement collection in appropriate file (e.g., `agent/system.go`)
2. Add to `CombinedData` structure
3. Update caching logic if needed
4. Add tests

### Adding New Actions

1. Define action in `internal/common/common-ws.go`
2. Implement handler in agent
3. Add validation and security checks
4. Update hub to support new action

### Custom Monitoring

The agent can be extended with custom monitoring:

```go
// Example: Custom metric collection
func (a *Agent) CollectCustomMetric() (interface{}, error) {
    // Your collection logic here
    return customData, nil
}
```

## Testing

### Unit Tests

```bash
make test
```

### Integration Tests

Integration tests verify hub-agent communication:

```bash
go test -tags=testing ./internal/hub/...
```

### Manual Testing

1. Start a test agent:
   ```bash
   make dev-agent
   ```

2. Connect to test hub:
   ```bash
   export SONAR_AGENT_HUB_URL=ws://localhost:8090
   export SONAR_AGENT_TOKEN=test-token
   ```

## Debugging

### Enable Debug Logging

```bash
export SONAR_AGENT_LOG_LEVEL=debug
```

### Inspect Communication

Use Wireshark or similar tools to inspect WebSocket traffic:

```bash
# Capture WebSocket traffic
tcpdump -i any -w capture.pcap port 8090
```

### Profile Performance

```go
import _ "net/http/pprof"

// Add to agent startup
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## Future Enhancements

### Planned Features

- **Command execution**: Secure command execution framework
- **Service management**: Full service control
- **File operations**: Secure file management
- **Log aggregation**: Centralized log collection
- **Custom scripts**: User-defined monitoring scripts
- **Plugin system**: Extensible plugin architecture

### Phase Roadmap

- **Phase 0.1** (Current): Rename to SONAR, PostgreSQL, framework stubs
- **Phase 0.2**: Command execution implementation
- **Phase 0.3**: Service management implementation
- **Phase 0.4**: File operations implementation
- **Phase 1.0**: Full feature set with security hardening

## Contributing

We welcome contributions! Areas that need help:

- Platform-specific optimizations
- New metric collection
- Performance improvements
- Documentation
- Tests

## References

- **gopsutil**: https://github.com/shirou/gopsutil
- **NVML**: https://developer.nvidia.com/nvidia-management-library-nvml
- **Docker API**: https://docs.docker.com/engine/api/
- **CBOR**: https://cbor.io/
- **WebSocket**: https://datatracker.ietf.org/doc/html/rfc6455

## Support

For technical questions and architecture discussions:
- GitHub Issues: https://github.com/henrygd/beszel/issues
- GitHub Discussions: https://github.com/henrygd/beszel/discussions
