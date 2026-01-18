#!/bin/bash
# Script to reset SONAR development environment (removes all data)

set -e

echo "WARNING: This will delete all data (database, volumes, etc.)"
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo "Stopping and removing containers..."
docker-compose down -v

echo "Removing Docker images..."
docker rmi sonar-hub:latest sonar-agent:latest 2>/dev/null || true

echo "Starting fresh environment..."
./scripts/docker-up.sh

echo ""
echo "Development environment reset complete!"
echo ""
