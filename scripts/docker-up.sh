#!/bin/bash
# Script to start SONAR development environment with Docker Compose

set -e

echo "Starting SONAR development environment..."

# Check if .env file exists, if not copy from .env.example
if [ ! -f .env ]; then
    echo "Creating .env file from .env.example..."
    cp .env.example .env
    echo "Please edit .env file with your configuration before proceeding."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed"
    exit 1
fi

# Build and start services
echo "Building Docker images..."
docker-compose build

echo "Starting services..."
docker-compose up -d

echo ""
echo "Services started successfully!"
echo ""
echo "SONAR Hub: http://localhost:8090"
echo "PostgreSQL: localhost:5432"
echo ""
echo "View logs with: docker-compose logs -f"
echo "Stop services with: docker-compose down"
echo ""
