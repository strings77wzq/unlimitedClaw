#!/bin/bash
# Load testing script for unlimitedClaw gateway
# Usage: ./scripts/loadtest.sh [URL] [DURATION] [CONCURRENCY]

set -euo pipefail

URL="${1:-http://localhost:18790}"
DURATION="${2:-10}"
CONCURRENCY="${3:-10}"
TOTAL_REQUESTS=$((CONCURRENCY * 100))

echo "=== unlimitedClaw Load Test ==="
echo "Target: $URL"
echo "Duration: ${DURATION}s"
echo "Concurrency: $CONCURRENCY"
echo ""

echo "--- Health Check ---"
if ! curl -sf "$URL/health" > /dev/null 2>&1; then
    echo "ERROR: Gateway not responding at $URL/health"
    exit 1
fi
echo "Gateway is healthy"
echo ""

echo "--- Test 1: Health Endpoint ---"
start=$(date +%s%N)
for i in $(seq 1 $TOTAL_REQUESTS); do
    curl -sf "$URL/health" > /dev/null &
    if [ $((i % CONCURRENCY)) -eq 0 ]; then
        wait
    fi
done
wait
end=$(date +%s%N)
elapsed=$(( (end - start) / 1000000 ))
qps=$((TOTAL_REQUESTS * 1000 / elapsed))
echo "Requests: $TOTAL_REQUESTS | Time: ${elapsed}ms | QPS: ~$qps"
echo ""

echo "--- Test 2: Chat Endpoint ---"
start=$(date +%s%N)
for i in $(seq 1 $TOTAL_REQUESTS); do
    curl -sf -X POST "$URL/api/chat" \
        -H "Content-Type: application/json" \
        -d '{"message":"test","session_id":"bench"}' > /dev/null &
    if [ $((i % CONCURRENCY)) -eq 0 ]; then
        wait
    fi
done
wait
end=$(date +%s%N)
elapsed=$(( (end - start) / 1000000 ))
qps=$((TOTAL_REQUESTS * 1000 / elapsed))
echo "Requests: $TOTAL_REQUESTS | Time: ${elapsed}ms | QPS: ~$qps"
echo ""

echo "=== Load Test Complete ==="
