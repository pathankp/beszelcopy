# SONAR Agent Setup Guide

This guide covers installing and configuring the SONAR agent on your systems.

## Overview

The SONAR agent is a lightweight monitoring daemon that runs on each system you want to monitor. It collects system metrics and sends them to the SONAR hub.

## Prerequisites

- A running SONAR hub instance
- Root/administrator access on the target system
- Network connectivity to the SONAR hub

## Installation Methods

### Method 1: Docker (Recommended)

The easiest way to run the SONAR agent is with Docker:

```bash
docker run -d \
  --name sonar-agent \
  --network host \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v ./sonar_agent_data:/var/lib/sonar-agent \
  -e SONAR_AGENT_LISTEN=45876 \
  -e SONAR_AGENT_KEY="your-public-key" \
  -e SONAR_AGENT_TOKEN="your-token" \
  -e SONAR_AGENT_HUB_URL="ws://your-hub:8090" \
  sonar-agent:latest
```

#### Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  sonar-agent:
    image: sonar-agent:latest
    container_name: sonar-agent
    restart: unless-stopped
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./sonar_agent_data:/var/lib/sonar-agent
    environment:
      SONAR_AGENT_LISTEN: 45876
      SONAR_AGENT_KEY: "your-public-key"
      SONAR_AGENT_TOKEN: "your-token"
      SONAR_AGENT_HUB_URL: "ws://your-hub:8090"
```

Then run:

```bash
docker-compose up -d
```

### Method 2: Binary Installation (Linux)

1. Download the latest agent binary from the releases page

2. Install the binary:

```bash
sudo install -m 755 sonar-agent /usr/local/bin/sonar-agent
```

3. Create systemd service:

```bash
sudo cp systemd/sonar-agent.service /etc/systemd/system/
```

4. Create environment file:

```bash
sudo mkdir -p /etc/sonar-agent
sudo nano /etc/sonar-agent/env
```

Add your configuration:

```
SONAR_AGENT_LISTEN=45876
SONAR_AGENT_KEY=your-public-key
SONAR_AGENT_TOKEN=your-token
SONAR_AGENT_HUB_URL=ws://your-hub:8090
```

5. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable sonar-agent
sudo systemctl start sonar-agent
```

6. Check status:

```bash
sudo systemctl status sonar-agent
```

### Method 3: Binary Installation (Windows)

1. Download the Windows agent binary

2. Create a directory for the agent:

```powershell
New-Item -ItemType Directory -Path "C:\Program Files\SONAR Agent"
```

3. Copy the binary to the directory

4. Create a Windows service (requires administrator PowerShell):

```powershell
New-Service -Name "SONARAgent" `
  -BinaryPathName '"C:\Program Files\SONAR Agent\sonar-agent.exe"' `
  -DisplayName "SONAR Agent" `
  -Description "SONAR monitoring agent" `
  -StartupType Automatic
```

5. Set environment variables in the registry or use a configuration file

6. Start the service:

```powershell
Start-Service SONARAgent
```

## Configuration

### Getting Credentials

1. Log in to your SONAR hub web interface
2. Navigate to Settings → Systems
3. Click "Add System"
4. Copy the public key and token provided

### Environment Variables

The agent can be configured using environment variables:

#### Required Variables

- `SONAR_AGENT_KEY`: Public key from the hub (required for SSH connection)
- `SONAR_AGENT_TOKEN`: Authentication token from the hub
- `SONAR_AGENT_HUB_URL`: WebSocket URL of the hub (e.g., `ws://hub.example.com:8090`)

#### Optional Variables

- `SONAR_AGENT_LISTEN`: Port to listen on (default: 45876)
- `SONAR_AGENT_LOG_LEVEL`: Log level (debug, info, warn, error)
- `SONAR_AGENT_MEM_CALC`: Memory calculation formula
- `SONAR_AGENT_DISK_USAGE_CACHE`: Cache disk usage for sleeping disks (e.g., "15m")

### Configuration File

You can also use a YAML configuration file. See `agent/config.example.yml` for all options.

## Monitoring Additional Filesystems

To monitor additional filesystems (e.g., external drives), mount them in the container:

```bash
docker run -d \
  --name sonar-agent \
  --network host \
  -v /mnt/disk1/.sonar:/extra-filesystems/disk1:ro \
  -v /mnt/disk2/.sonar:/extra-filesystems/disk2:ro \
  -e SONAR_AGENT_KEY="your-key" \
  -e SONAR_AGENT_TOKEN="your-token" \
  -e SONAR_AGENT_HUB_URL="ws://your-hub:8090" \
  sonar-agent:latest
```

Create the `.sonar` directories on your disks:

```bash
sudo mkdir -p /mnt/disk1/.sonar
sudo mkdir -p /mnt/disk2/.sonar
```

## Network Requirements

### Firewall Rules

The agent needs to:
- **Receive** connections on the configured port (default: 45876)
- **Initiate** WebSocket connections to the hub

If using a firewall, allow incoming connections on the agent port:

```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 45876/tcp

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=45876/tcp
sudo firewall-cmd --reload
```

### Port Forwarding

If the agent is behind NAT, configure port forwarding on your router:
- External port: 45876 (or your chosen port)
- Internal IP: Agent's IP address
- Internal port: 45876 (or your chosen port)

## Docker Monitoring

To monitor Docker containers, the agent needs access to the Docker socket:

```bash
-v /var/run/docker.sock:/var/run/docker.sock:ro
```

This is **read-only** (`ro`) for security.

## GPU Monitoring

### NVIDIA GPUs

For NVIDIA GPU monitoring, use the NVIDIA variant of the agent image:

```bash
docker run -d \
  --name sonar-agent \
  --network host \
  --runtime=nvidia \
  -e NVIDIA_VISIBLE_DEVICES=all \
  -e SONAR_AGENT_KEY="your-key" \
  -e SONAR_AGENT_TOKEN="your-token" \
  -e SONAR_AGENT_HUB_URL="ws://your-hub:8090" \
  sonar-agent:nvidia
```

### AMD/Intel GPUs

AMD and Intel GPU monitoring is supported on Linux with appropriate drivers.

## Security Considerations

1. **Keep credentials secure**: Never commit `SONAR_AGENT_KEY` or `SONAR_AGENT_TOKEN` to version control
2. **Use read-only mounts**: Mount Docker socket as read-only (`:ro`)
3. **Network isolation**: Consider using a VPN or private network for hub-agent communication
4. **Regular updates**: Keep the agent updated to receive security patches

## Troubleshooting

### Agent Won't Start

1. Check logs:
   ```bash
   # Docker
   docker logs sonar-agent
   
   # Systemd
   sudo journalctl -u sonar-agent -f
   ```

2. Verify environment variables are set correctly
3. Check network connectivity to the hub

### Agent Not Appearing in Hub

1. Verify `SONAR_AGENT_KEY` and `SONAR_AGENT_TOKEN` match the hub
2. Check firewall rules
3. Verify `SONAR_AGENT_HUB_URL` is correct
4. Check hub logs for connection attempts

### Docker Container Monitoring Not Working

1. Verify Docker socket is mounted: `-v /var/run/docker.sock:/var/run/docker.sock:ro`
2. Check Docker socket permissions
3. Restart the agent container

### High CPU/Memory Usage

1. Check log level (set to `info` or `warn` instead of `debug`)
2. Increase cache durations if monitoring many filesystems
3. Review monitored containers and filesystems

## Updating the Agent

### Docker

```bash
docker pull sonar-agent:latest
docker stop sonar-agent
docker rm sonar-agent
# Run with new image
docker run -d ...
```

### Binary

1. Download new binary
2. Stop the service
3. Replace the binary
4. Start the service

```bash
sudo systemctl stop sonar-agent
sudo install -m 755 sonar-agent /usr/local/bin/sonar-agent
sudo systemctl start sonar-agent
```

## Uninstalling

### Docker

```bash
docker stop sonar-agent
docker rm sonar-agent
docker rmi sonar-agent:latest
```

### Binary (Linux)

```bash
sudo systemctl stop sonar-agent
sudo systemctl disable sonar-agent
sudo rm /etc/systemd/system/sonar-agent.service
sudo rm /usr/local/bin/sonar-agent
sudo rm -rf /etc/sonar-agent
sudo systemctl daemon-reload
```

## Advanced Configuration

For advanced configuration options, see the [Agent Architecture](AGENT_ARCHITECTURE.md) documentation.

## Support

For issues and questions:
- GitHub Issues: https://github.com/henrygd/beszel/issues
- GitHub Discussions: https://github.com/henrygd/beszel/discussions
