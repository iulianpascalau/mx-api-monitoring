#!/bin/bash

# Configuration
APP_NAME="mx-api-monitoring-agent"
APP_DIR="/home/${USER}/mx-api-monitoring/services/agent"
EXEC_PATH="${APP_DIR}/agent"

# Create the service file content
SERVICE_CONTENT="[Unit]
Description=Mvx API monitoring Agent Go service
After=network-online.target

[Service]
User=${USER}
WorkingDirectory=${APP_DIR}
ExecStart=${EXEC_PATH} -log-save -log-level *:DEBUG
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
"

# Path to the systemd service file
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

# Write the service file
echo "Creating systemd service file at ${SERVICE_FILE}..."
sudo bash -c "echo '${SERVICE_CONTENT}' > ${SERVICE_FILE}"

# Reload systemd daemon
echo "Reloading systemd daemon..."
sudo systemctl daemon-reload

# Enable the service
echo "Enabling ${APP_NAME} service..."
sudo systemctl enable ${APP_NAME}

# Start the service
echo "Starting ${APP_NAME} service..."
sudo systemctl start ${APP_NAME}

# Show status
echo "Service status:"
sudo systemctl status ${APP_NAME} --no-pager
