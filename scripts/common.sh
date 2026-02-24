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

# Function to ensure the correct Node and Yarn versions are installed
ensure_node_yarn_installed() {
    # Load variables if not already loaded (and if the file exists)
    if [ -f ./scripts/variables.cfg ]; then
        source ./scripts/variables.cfg
    fi

    if [ -z "$NODE_LATEST_TESTED" ]; then
        echo "NODE_LATEST_TESTED not set, defaulting to 22.12.0"
        NODE_LATEST_TESTED="22.12.0"
    fi

    get_node_ver() {
        if command -v node &> /dev/null; then
            node -v | sed 's/^v//'
        else
            echo "none"
        fi
    }

    CURRENT_NODE_VER=$(get_node_ver)

    if [[ "$CURRENT_NODE_VER" != "$NODE_LATEST_TESTED"* ]]; then
        echo "Node version mismatch (Current: $CURRENT_NODE_VER, Required: $NODE_LATEST_TESTED). Installing..."
        
        ARCH=$(uname -m)
        if [ "$ARCH" == "x86_64" ]; then
            NODE_ARCH="x64"
        elif [ "$ARCH" == "aarch64" ]; then
            NODE_ARCH="arm64"
        else
            echo "Unsupported architecture: $ARCH"
            exit 1
        fi

        wget -q "https://nodejs.org/dist/v${NODE_LATEST_TESTED}/node-v${NODE_LATEST_TESTED}-linux-${NODE_ARCH}.tar.xz" -O /tmp/node.tar.xz
        if [ $? -ne 0 ]; then
            echo "Error: Failed to download Node $NODE_LATEST_TESTED"
            exit 1
        fi
        
        sudo mkdir -p /usr/local/lib/nodejs
        sudo tar -xJvf /tmp/node.tar.xz -C /usr/local/lib/nodejs
        NODE_DIR=$(ls -d /usr/local/lib/nodejs/node-v${NODE_LATEST_TESTED}-linux-${NODE_ARCH})
        
        # Link binaries to /usr/local/bin
        sudo ln -sf "${NODE_DIR}/bin/node" /usr/local/bin/node
        sudo ln -sf "${NODE_DIR}/bin/npm" /usr/local/bin/npm
        sudo ln -sf "${NODE_DIR}/bin/npx" /usr/local/bin/npx
        
        rm /tmp/node.tar.xz
        echo "Node $NODE_LATEST_TESTED installed."
    else
        echo "Node version $CURRENT_NODE_VER matches required version."
    fi

    # Find the current node directory to export bin path
    ARCH=$(uname -m)
    if [ "$ARCH" == "x86_64" ]; then NODE_ARCH="x64"; elif [ "$ARCH" == "aarch64" ]; then NODE_ARCH="arm64"; fi
    NODE_BIN_DIR=$(ls -d /usr/local/lib/nodejs/node-v${NODE_LATEST_TESTED}-linux-${NODE_ARCH}/bin 2>/dev/null || echo "")
    
    if [ -n "$NODE_BIN_DIR" ]; then
        export PATH="$NODE_BIN_DIR:$PATH"
    fi

    # Ensure Yarn is installed
    if ! command -v yarn &> /dev/null; then
        echo "Yarn not found. Installing globally via npm..."
        sudo /usr/local/bin/npm install -g yarn
        
        # After installation, we might need to symlink it if not already in /usr/local/bin
        if [ -f "${NODE_BIN_DIR}/yarn" ] && [ ! -f "/usr/local/bin/yarn" ]; then
            sudo ln -sf "${NODE_BIN_DIR}/yarn" /usr/local/bin/yarn
        fi
    else
        echo "Yarn $(yarn -v) found."
    fi
}
