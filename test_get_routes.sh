#!/bin/bash

# Target API Gateway base endpoint address
GATEWAY_URL="https://localhost:8443"

# List of target GET endpoints with varying params and weights to test cache segregation
ROUTES=(
    "/home"
    "/slow-home"
    "/api/users"
    "/api/users?role=admin"
    "/api/users?role=developer"
    "/api/users/tarun"
    "/api/users/guest_user"
    "/api/files/sample.txt"
)

# Total requests to send across the cluster test array
TOTAL_REQUESTS=1000

echo "========================================================="
echo "🚀 Initializing Caching Gateway GET Test Suite"
echo "🎯 Target URL: $GATEWAY_URL"
echo "🔢 Total Parallel Operations: $TOTAL_REQUESTS"
echo "========================================================="

# Ensure an upload mock file exists so the download route doesn't throw a flat 404
mkdir -p uploads
echo "LB4A Gateway Performance Testing File Content Check" > uploads/sample.txt

# --- PATCHED FOR LOOP SYNTAX ---
# Modified section inside your test_get_routes.sh loop
for ((i=1; i<=TOTAL_REQUESTS; i++)); do
    RANDOM_INDEX=$(( RANDOM % ${#ROUTES[@]} ))
    TARGET_ROUTE=${ROUTES[$RANDOM_INDEX]}
    
    # -sI: Silent Head-only fetch. Extracts just the response headers.
    # grep: Isolates the specific headers we want to monitor in real-time.
    echo -e "\n🔄 Path: $TARGET_ROUTE"
    curl -k -sI "$GATEWAY_URL$TARGET_ROUTE" | grep -E "HTTP/|X-Cache:|Content-Type:" &
    
    sleep 0.05 # Slightly increased sleep so you can actually read the output scrolling
done
wait

echo "📥 All background workers spawned. Awaiting network stream terminations..."
wait
echo "✅ Test run complete. Inspect your gateway logs or invoke your log analyzer script."
