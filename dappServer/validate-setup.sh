#!/bin/bash

# Setup Validation Script
# This script checks if your local environment is ready for testing

echo "════════════════════════════════════════════════════════════"
echo "🔍 Queue System - Setup Validation"
echo "════════════════════════════════════════════════════════════"
echo ""

ERRORS=0
WARNINGS=0

# Check 1: Configuration files
echo "📁 Checking configuration files..."
if [ -f ".config/config.toml" ]; then
    echo "   ✅ config.toml exists"
else
    echo "   ❌ config.toml NOT FOUND"
    echo "      Run: cp .config/config.toml.example .config/config.toml"
    ERRORS=$((ERRORS + 1))
fi

if [ -f ".config/.env" ]; then
    echo "   ✅ .env exists"
else
    echo "   ❌ .env NOT FOUND"
    echo "      Run: cp .config/.env.example .config/.env"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 2: Binary exists
echo "🔨 Checking compiled binary..."
if [ -f "./dapp-server" ]; then
    echo "   ✅ dapp-server binary exists"
else
    echo "   ❌ dapp-server binary NOT FOUND"
    echo "      Run: go build -o dapp-server"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 3: Rubix nodes
echo "🌐 Checking Rubix nodes..."
if pgrep -f "rubix" > /dev/null; then
    echo "   ✅ Rubix process(es) running"
    NODE_COUNT=$(pgrep -f "rubix" | wc -l)
    echo "      Found $NODE_COUNT Rubix node(s)"
else
    echo "   ⚠️  No Rubix processes found"
    echo "      You may need to start your Rubix nodes"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Check 4: Node connectivity (if config exists)
if [ -f ".config/config.toml" ]; then
    echo "🔌 Checking node connectivity..."

    # Extract ports from config
    PORTS=$(grep "port = " .config/config.toml | sed 's/.*"\([0-9]*\)".*/\1/')

    for PORT in $PORTS; do
        if curl -s -f "http://localhost:$PORT/api/node-status" > /dev/null 2>&1; then
            echo "   ✅ Node on port $PORT is accessible"
        else
            echo "   ❌ Node on port $PORT is NOT accessible"
            echo "      Is the Rubix node running on port $PORT?"
            WARNINGS=$((WARNINGS + 1))
        fi
    done
    echo ""
fi

# Check 5: Contract hash in .env
if [ -f ".config/.env" ]; then
    echo "📜 Checking smart contract configuration..."

    if grep -q "TRANSFER_CONTRACT=Qm" .config/.env 2>/dev/null; then
        HASH=$(grep "TRANSFER_CONTRACT=" .config/.env | cut -d'=' -f2)
        if [ "$HASH" = "QmYourTransferContractHashHere" ]; then
            echo "   ⚠️  TRANSFER_CONTRACT is still placeholder"
            echo "      You need to deploy a contract and update .env"
            WARNINGS=$((WARNINGS + 1))
        else
            echo "   ✅ TRANSFER_CONTRACT is set: $HASH"
        fi
    else
        echo "   ⚠️  TRANSFER_CONTRACT not found or invalid in .env"
        echo "      You need to deploy a contract and add the hash"
        WARNINGS=$((WARNINGS + 1))
    fi
    echo ""
fi

# Check 6: Port 9000 availability
echo "🔓 Checking port availability..."
if lsof -i:9000 > /dev/null 2>&1; then
    echo "   ⚠️  Port 9000 is already in use"
    echo "      Another process may be using it"
    WARNINGS=$((WARNINGS + 1))
else
    echo "   ✅ Port 9000 is available"
fi
echo ""

# Summary
echo "════════════════════════════════════════════════════════════"
echo "📊 Validation Summary"
echo "════════════════════════════════════════════════════════════"
echo "   Errors:   $ERRORS"
echo "   Warnings: $WARNINGS"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo "✅ ALL CHECKS PASSED!"
    echo ""
    echo "You're ready to start the server:"
    echo "   ./dapp-server"
    echo ""
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo "⚠️  READY WITH WARNINGS"
    echo ""
    echo "You can start the server, but some features may not work:"
    echo "   ./dapp-server"
    echo ""
    echo "Address warnings above for full functionality."
    echo ""
    exit 0
else
    echo "❌ SETUP INCOMPLETE"
    echo ""
    echo "Please fix the errors above before starting the server."
    echo ""
    echo "Quick fixes:"
    echo "   1. cp .config/config.toml.example .config/config.toml"
    echo "   2. cp .config/.env.example .config/.env"
    echo "   3. nano .config/config.toml  # Update DIDs"
    echo "   4. go build -o dapp-server"
    echo ""
    echo "See QUICK_START.md for detailed instructions."
    echo ""
    exit 1
fi
