#!/bin/bash

# Configuration: Update these to match your new TLS setup
BASE_URL="https://localhost:8443"
CURL_OPTS="-k -s" # -k bypasses certificate verification for local testing

echo "================================================="
echo "🚀 INITIATING LB4A GATEWAY STRESS TESTS (HTTPS)"
echo "================================================="

echo -e "\n[TEST 1] POST: Create user 'tarun' (JSON Payload)"
curl $CURL_OPTS -X POST "$BASE_URL/api/users" \
     -H "Content-Type: application/json" \
     -d '{"user": "tarun", "role": "admin", "status": "active"}'
echo ""

echo -e "\n[TEST 2] POST: Create user 'johndoe' (JSON Payload)"
curl $CURL_OPTS -X POST "$BASE_URL/api/users" \
     -H "Content-Type: application/json" \
     -d '{"user": "johndoe", "role": "guest", "status": "active"}'
echo ""

echo -e "\n[TEST 3] GET (Dynamic Route): Fetch ONLY 'tarun'"
curl $CURL_OPTS -X GET "$BASE_URL/api/users/tarun"
echo ""

echo -e "\n[TEST 4] GET (Query Params): Fetch ONLY admins"
curl $CURL_OPTS -X GET "$BASE_URL/api/users?role=admin"
echo ""

echo -e "\n[TEST 5] DELETE (Timeout Check): Delete 'tarun'"
# If your Gateway timeout is 500ms and backend takes 600ms, 
# you should see a '504 Gateway Timeout' or '502 Bad Gateway' here.
curl $CURL_OPTS -i -X DELETE "$BASE_URL/api/users/tarun"
echo ""

echo "================================================="
echo "🚀 INITIATING BINARY / FILE SYSTEM GATEWAY TESTS"
echo "================================================="

# 1. Create dummy files on the fly
echo "Creating dummy files..."
echo "Hello from File 1! This data passed through Go via TLS." > test1.txt
echo "Hello from File 2! Aegis Engine is working." > test2.txt

echo -e "\n[TEST 6] POST: Multi-File Upload (Multipart)"
curl $CURL_OPTS -X POST "$BASE_URL/api/files/upload" \
  -F "files=@test1.txt" \
  -F "files=@test2.txt"

echo -e "\n\n[TEST 7] GET: Download 'test1.txt' back through the Gateway"
curl $CURL_OPTS -X GET "$BASE_URL/api/files/test1.txt"

echo -e "\n\n[CLEANUP] Deleting local dummy files..."
rm test1.txt test2.txt

echo -e "\n================================================="
echo "✅ HTTPS TESTS COMPLETE"
echo "================================================="
