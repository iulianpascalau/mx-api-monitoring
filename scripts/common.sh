#!/bin/bash

# Function to ensure the correct Go version is installed
ensure_go_installed() {
    # Load variables if not already loaded (and if the file exists)
    if [ -f ./scripts/variables.cfg ]; then
        source ./scripts/variables.cfg
    fi

    if [ -z "$GO_LATEST_TESTED" ]; then
        echo "GO_LATEST_TESTED not set, defaulting to 1.24.11"
        GO_LATEST_TESTED="1.24.11"
    fi

    get_go_ver() {
        if command -v go &> /dev/null; then
            go version | awk '{print $3}' | sed 's/^go//'
        elif [ -f "/usr/local/go/bin/go" ]; then
            /usr/local/go/bin/go version | awk '{print $3}' | sed 's/^go//'
        else
            echo "none"
        fi
    }

    CURRENT_GO_VER=$(get_go_ver)

    if [ "$CURRENT_GO_VER" != "$GO_LATEST_TESTED" ]; then
        echo "Go version mismatch (Current: $CURRENT_GO_VER, Required: $GO_LATEST_TESTED). Installing..."
        
        ARCH=$(dpkg --print-architecture)
        wget -q "https://go.dev/dl/go${GO_LATEST_TESTED}.linux-${ARCH}.tar.gz" -O /tmp/go.tar.gz
        if [ $? -ne 0 ]; then
            echo "Error: Failed to download Go $GO_LATEST_TESTED"
            exit 1
        fi
        
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf /tmp/go.tar.gz
        rm /tmp/go.tar.gz
        
        export PATH=$PATH:/usr/local/go/bin
        echo "Go $GO_LATEST_TESTED installed."
    else
        echo "Go version $CURRENT_GO_VER matches required version."
        if ! command -v go &> /dev/null && [ -f "/usr/local/go/bin/go" ]; then
            export PATH=$PATH:/usr/local/go/bin
        fi
    fi
}
