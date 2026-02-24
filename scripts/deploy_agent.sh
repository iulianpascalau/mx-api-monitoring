#!/bin/bash
set -e

# Configuration
PROJECT_DIR="/home/${USER}/mx-api-monitoring"
AGENT_SERVICE="mx-api-monitoring-agent"

# Check argument
BRANCH=$1
if [ -z "$BRANCH" ]; then
    echo "Usage: $0 <branch_or_tag>"
    echo "Example: $0 main"
    exit 1
fi

echo "=========================================="
echo "Starting deployment for branch: $BRANCH"
echo "=========================================="

# Navigate to project directory
if [ ! -d "$PROJECT_DIR" ]; then
    echo "Error: Project directory $PROJECT_DIR does not exist."
    exit 1
fi
cd "$PROJECT_DIR"

# 1. Stop Services
echo "Step 1: Stopping services..."
sudo systemctl stop $AGENT_SERVICE || echo "Services not found or not running, skipping stop."

# 2. Checkout Code
echo "Step 2: Checking out code..."
git fetch --all
git checkout "$BRANCH"
git pull origin "$BRANCH"

# 3. Recompile Backend
echo "Step 3: Recompiling Backend..."
# Load common functions
source ./scripts/common.sh

# Ensure Go is installed
ensure_go_installed
GO_CMD="go"

cd ./services/agent
$GO_CMD build -v -ldflags="-X main.appVersion=$(git describe --tags --long --dirty)" -o agent main.go
if [ $? -ne 0 ]; then
    echo "Agent binary build failed!"
    exit 1
fi
echo "Agent binary build successful."

# 4. Restart Services
echo "Step 4: Restarting agent service..."

# Backend
if systemctl cat $AGENT_SERVICE > /dev/null 2>&1; then
    sudo systemctl start $AGENT_SERVICE
else
    echo "Service $AGENT_SERVICE not found. Creating it..."
    chmod +x "$PROJECT_DIR/scripts/create_agent_service.sh"
    "$PROJECT_DIR/scripts/create_agent_service.sh"
fi

# 5. Monitor
echo "Step 5: Monitoring status..."
sleep 5

if systemctl is-active --quiet $AGENT_SERVICE; then
    echo "✅ $AGENT_SERVICE is active."
else
    echo "❌ $AGENT_SERVICE failed to start."
    sudo journalctl -u $AGENT_SERVICE -n 20 --no-pager
    exit 1
fi

echo "=========================================="
echo "Deployment Finished Successfully!"
echo "=========================================="
