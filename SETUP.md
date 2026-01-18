# SONAR Local Development Setup

This guide will help you set up a local development environment for SONAR.

## Prerequisites

- Docker and Docker Compose
- Go 1.25 or later (for local development without Docker)
- Node.js/Bun (for frontend development)
- PostgreSQL 16+ (if not using Docker)

## Quick Start with Docker

The easiest way to get started is using Docker Compose:

```bash
# 1. Clone the repository
git clone https://github.com/henrygd/beszel.git
cd beszel

# 2. Copy environment file
cp .env.example .env

# 3. Edit .env file with your configuration
# At minimum, set SONAR_HUB_POSTGRES_PASSWORD

# 4. Start the development environment
./scripts/docker-up.sh

# Or manually:
docker-compose up -d
```

The SONAR hub will be available at http://localhost:8090

Default credentials:
- Email: admin@sonar.dev
- Password: admin123

## Manual Setup (Without Docker)

### 1. PostgreSQL Setup

Install and start PostgreSQL:

```bash
# On Ubuntu/Debian
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql

# On macOS
brew install postgresql@16
brew services start postgresql@16
```

Create database and user:

```sql
CREATE DATABASE sonar;
CREATE USER sonar WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE sonar TO sonar;
```

### 2. Environment Configuration

Create a `.env` file or export environment variables:

```bash
export SONAR_HUB_POSTGRES_HOST=localhost
export SONAR_HUB_POSTGRES_PORT=5432
export SONAR_HUB_POSTGRES_DB=sonar
export SONAR_HUB_POSTGRES_USER=sonar
export SONAR_HUB_POSTGRES_PASSWORD=your_password
export SONAR_HUB_APP_URL=http://localhost:8090
export SONAR_HUB_USER_EMAIL=admin@sonar.dev
export SONAR_HUB_USER_PASSWORD=admin123
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Build and Run

#### Hub

```bash
# Build frontend (optional, can skip for development)
make build-web-ui

# Run hub in development mode
make dev-hub
```

#### Agent

```bash
# Run agent in development mode
make dev-agent
```

## Development Workflow

### Frontend Development

Start the frontend development server:

```bash
make dev-server
```

This will start the Vite development server at http://localhost:5173

### Hub Development

Run the hub in development mode:

```bash
make dev-hub
```

This watches for Go file changes and automatically restarts the hub.

### Full Stack Development

Run all services simultaneously:

```bash
make dev
```

This starts:
- Frontend dev server (port 5173)
- Hub backend (port 8090)
- Agent

## Database Management

### Migrations

Migrations run automatically when the hub starts. The database schema is created on first run.

### Reset Database

To reset the database:

```bash
# With Docker
./scripts/docker-reset.sh

# Manually
docker-compose down -v
docker-compose up -d
```

## Docker Commands

```bash
# Build images
make docker-build

# Start services
make docker-up

# Stop services
make docker-down

# View logs
make docker-logs

# Reset everything (WARNING: deletes all data)
make docker-reset
```

## Project Structure

```
.
├── agent/              # Agent code
├── internal/
│   ├── cmd/           # Command-line interfaces
│   ├── db/            # PostgreSQL database layer
│   ├── hub/           # Hub core logic
│   ├── site/          # Frontend React application
│   └── ...
├── systemd/           # Systemd service files
├── scripts/           # Helper scripts
├── Dockerfile.hub     # Hub Docker image
├── Dockerfile.agent   # Agent Docker image
└── docker-compose.yml # Development environment
```

## Environment Variables

See `.env.example` for all available environment variables.

### Hub Environment Variables

- `SONAR_HUB_POSTGRES_*`: PostgreSQL connection settings
- `SONAR_HUB_APP_URL`: Public URL of the hub
- `SONAR_HUB_USER_EMAIL`: Initial admin email
- `SONAR_HUB_USER_PASSWORD`: Initial admin password
- `SONAR_HUB_DISABLE_PASSWORD_AUTH`: Disable password authentication
- `SONAR_HUB_USER_CREATION`: Allow user creation via OAuth
- `SONAR_HUB_MFA_OTP`: Enable MFA/OTP
- `SONAR_HUB_SHARE_ALL_SYSTEMS`: Share systems across all users
- `SONAR_HUB_CONTAINER_DETAILS`: Enable container details endpoints

### Agent Environment Variables

- `SONAR_AGENT_LISTEN`: Port to listen on (default: 45876)
- `SONAR_AGENT_KEY`: Public key from hub
- `SONAR_AGENT_TOKEN`: Authentication token from hub
- `SONAR_AGENT_HUB_URL`: Hub WebSocket URL
- `SONAR_AGENT_LOG_LEVEL`: Log level (debug, info, warn, error)

## Troubleshooting

### Database Connection Issues

1. Check PostgreSQL is running: `systemctl status postgresql`
2. Verify connection settings in `.env`
3. Check PostgreSQL logs: `docker-compose logs postgres`

### Hub Not Starting

1. Check logs: `docker-compose logs hub`
2. Verify environment variables are set correctly
3. Ensure PostgreSQL is healthy: `docker-compose ps`

### Agent Not Connecting

1. Check agent logs: `docker-compose logs agent`
2. Verify `SONAR_AGENT_KEY` and `SONAR_AGENT_TOKEN` are correct
3. Ensure hub is accessible at `SONAR_AGENT_HUB_URL`

## Testing

Run tests:

```bash
make test
```

Run linter:

```bash
make lint
```

## Building for Production

```bash
# Build hub
make build-hub

# Build agent
make build-agent

# Build both
make build
```

Executables will be in the `build/` directory.

## Additional Resources

- [Agent Setup Guide](AGENT_SETUP.md)
- [Agent Architecture](AGENT_ARCHITECTURE.md)
- [README](readme.md)

## Support

For issues and questions:
- GitHub Issues: https://github.com/henrygd/beszel/issues
- GitHub Discussions: https://github.com/henrygd/beszel/discussions
