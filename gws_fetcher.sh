#!/bin/bash

mkdir -p bin_update && cd bin_update

# 1. Update macOS Intel (amd64)
curl -sLO https://github.com/googleworkspace/cli/releases/download/v0.22.5/google-workspace-cli-x86_64-apple-darwin.tar.gz
tar -xzf google-workspace-cli-x86_64-apple-darwin.tar.gz && mv gws ../bin/gws-darwin-amd64

# 2. Update macOS Apple Silicon (arm64)
curl -sLO https://github.com/googleworkspace/cli/releases/download/v0.22.5/google-workspace-cli-aarch64-apple-darwin.tar.gz
tar -xzf google-workspace-cli-aarch64-apple-darwin.tar.gz && mv gws ../bin/gws-darwin-arm64

# 3. Update Linux (amd64)
curl -sLO https://github.com/googleworkspace/cli/releases/download/v0.22.5/google-workspace-cli-x86_64-unknown-linux-gnu.tar.gz
tar -xzf google-workspace-cli-x86_64-unknown-linux-gnu.tar.gz && mv gws ../bin/gws-linux-amd64

# 4. Update Windows (amd64)
curl -sLO https://github.com/googleworkspace/cli/releases/download/v0.22.5/google-workspace-cli-x86_64-pc-windows-msvc.zip
unzip -q google-workspace-cli-x86_64-pc-windows-msvc.zip && mv gws.exe ../bin/gws-windows-amd64.exe

cd .. && rm -rf bin_update
echo "Successfully updated gws binaries!"

