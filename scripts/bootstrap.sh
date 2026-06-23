#!/bin/bash

set -e

echo "🚀 Bootstrapping Kiyoshi..."

echo "📦 Installing root dependencies..."
npm install

echo "📦 Installing CLI dependencies..."
cd apps/cli
go mod download
go mod tidy
cd ../..

echo "📦 Installing web dependencies..."
cd apps/web
npm install
cd ../..

echo "✅ Bootstrap complete!"
echo ""
echo "🎯 Next steps:"
echo "  1. Start CLI: cd apps/cli && go run main.go"
echo "  2. Start Web: cd apps/web && npm run dev"
echo ""
echo "📖 Documentation: https://github.com/romero429-collab/kiyoshi/docs"
