#!/bin/bash
set -e

# Configuration
PROJECT_DIR="/home/${USER}/api-monitoring"
AGGREGATION_SERVICE="api-monitoring-aggregation"
FRONTEND_DIR="${PROJECT_DIR}/frontend"
# Load common functions
source "$(dirname "$0")/common.sh"

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
sudo systemctl stop $AGGREGATION_SERVICE || echo "Services not found or not running, skipping stop."

# 2. Checkout Code
echo "Step 2: Checking out code..."
git fetch --all
git checkout "$BRANCH"
git pull origin "$BRANCH"

# 3. Build Frontend
echo "Step 3: Building Frontend..."
ensure_node_yarn_installed
if [ -d "$FRONTEND_DIR" ]; then
    cd "$FRONTEND_DIR"
    # Ensure dependencies are installed
    yarn install --ignore-engines
    # Build for web
    npx expo export --platform web
    echo "Frontend build successful."
    cd "$PROJECT_DIR"
else
    echo "Warning: Frontend directory not found at $FRONTEND_DIR, skipping frontend build."
fi

# 4. Recompile Backend
echo "Step 4: Recompiling Backend..."

# Ensure Go and Node are installed
ensure_go_installed
GO_CMD="go"

cd ./services/aggregation
# Build the aggregation binary with limited parallelism to avoid OOM/hangs on small VMs
$GO_CMD build -p 1 -v -ldflags="-X main.appVersion=$(git describe --tags --long --dirty)" -o aggregation main.go
if [ $? -ne 0 ]; then
    echo "Aggregation binary build failed!"
    exit 1
fi
echo "Aggregation binary build successful."
cd "$PROJECT_DIR"

# 5. Restart Services
echo "Step 5: Restarting aggregation service..."

if systemctl cat $AGGREGATION_SERVICE > /dev/null 2>&1; then
    sudo systemctl start $AGGREGATION_SERVICE
else
    echo "Service $AGGREGATION_SERVICE not found. Creating it..."
    chmod +x "$PROJECT_DIR/scripts/create_aggregation_service.sh"
    "$PROJECT_DIR/scripts/create_aggregation_service.sh"
fi

# 6. Monitor
echo "Step 6: Monitoring status..."
sleep 5

if systemctl is-active --quiet $AGGREGATION_SERVICE; then
    echo "✅ $AGGREGATION_SERVICE is active."
else
    echo "❌ $AGGREGATION_SERVICE failed to start."
    sudo journalctl -u $AGGREGATION_SERVICE -n 20 --no-pager
    exit 1
fi

echo "=========================================="
echo "Full Deployment Finished Successfully!"
echo "=========================================="

