#!/bin/bash

# WACLI API Server Startup Script

# Check if API keys are set
if [ -z "$WACLI_API_KEYS" ]; then
    echo "Error: WACLI_API_KEYS environment variable is not set"
    echo "Please export API keys before running: export WACLI_API_KEYS=your-secret-key"
    exit 1
fi

# Default values
HOST=${WACLI_API_HOST:-"0.0.0.0"}
PORT=${WACLI_API_PORT:-8080}
STORE_DIR=${WACLI_STORE_DIR:-"$HOME/.wacli"}
GIN_MODE=${GIN_MODE:-"debug"}

echo "======================================"
echo "Starting WACLI API Server"
echo "======================================"
echo "Host: $HOST"
echo "Port: $PORT"
echo "Store: $STORE_DIR"
echo "Mode: $GIN_MODE"
echo "======================================"
echo ""

# Create store directory if it doesn't exist
mkdir -p "$STORE_DIR"

# Run the server
exec ./bin/wacli-api
