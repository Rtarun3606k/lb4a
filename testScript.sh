#!/bin/bash

echo "================================================="
echo "🚀 INITIATING LB4A GATEWAY STRESS TESTS"
echo "================================================="

echo -e "\n[TEST 1] POST: Create user 'tarun' (JSON Payload)"
curl -s -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"user": "tarun", "role": "admin", "status": "active"}'
echo ""

echo -e "\n[TEST 2] POST: Create user 'johndoe' (JSON Payload)"
curl -s -X POST http://localhost:8080/api/users \
     -H "Content-Type: application/json" \
     -d '{"user": "johndoe", "role": "guest", "status": "active"}'
echo ""

echo -e "\n[TEST 3] GET (Dynamic Route): Fetch ONLY 'tarun'"
# Proves Gateway can handle /api/users/tarun without breaking
curl -s -X GET http://localhost:8080/api/users/tarun
echo ""

echo -e "\n[TEST 4] GET (Query Params): Fetch ONLY admins"
# Proves Gateway forwards the ?role=admin string to Python
curl -s -X GET "http://localhost:8080/api/users?role=admin"
echo ""

echo -e "\n[TEST 5] DELETE (Dynamic Route + Timeout Check): Delete 'tarun'"
# Takes 600ms. Will test if your Gateway enforces the 500ms timeout!
curl -s -X DELETE http://localhost:8080/api/users/tarun
echo ""


echo "================================================="
echo "🚀 INITIATING BINARY / FILE SYSTEM GATEWAY TESTS"
echo "================================================="

# 1. Create dummy files on the fly
echo "Creating dummy files..."
echo "Hello from File 1! This data passed through Go." > test1.txt
echo "Hello from File 2! Aegis Engine is working." > test2.txt

echo -e "\n[TEST 6] POST: Multi-File Upload"
# The -F flag tells curl to send multipart/form-data
curl -X POST http://localhost:8080/api/files/upload \
  -F "files=@test1.txt" \
  -F "files=@test2.txt"

echo -e "\n\n[TEST 7] GET: Download 'test1.txt' back through the Gateway"
# This tests if Go can stream a raw file back to the client
curl -X GET http://localhost:8080/api/files/test1.txt

echo -e "\n\n[CLEANUP] Deleting local dummy files..."
rm test1.txt test2.txt

echo -e "\n================================================="
echo "✅ TESTS COMPLETE"
echo "================================================="


