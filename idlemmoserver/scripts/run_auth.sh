#!/bin/bash

echo "Starting Auth Service..."

cd "$(dirname "$0")/.."

# 检查是否存在auth目录
if [ ! -d "auth" ]; then
    echo "Error: auth directory not found"
    exit 1
fi

cd auth

# 启动Auth Service
echo "Starting Auth Service on NATS..."
go run main.go