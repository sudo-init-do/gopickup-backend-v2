#!/bin/bash

# Configuration
API_URL="http://localhost:8080/api/v1"
EMAIL_CLIENT="client_$(date +%s)@test.com"
EMAIL_DRIVER="driver_$(date +%s)@test.com"
EMAIL_ADMIN="admin_$(date +%s)@test.com"
PASSWORD="password123"

echo "=== Testing Phase 3: Profiles + Admin Approval ==="

# 1. Register Client
echo -e "\n1. Registering Client..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_CLIENT\", \"password\": \"$PASSWORD\", \"role\": \"client\"}" | grep "message"

# 2. Login Client to get Token
echo -e "\n2. Logging in Client..."
# Force verify user in DB because we can't get OTP easily in script
sqlite3 gopickup.db "UPDATE users SET is_verified=1 WHERE email='$EMAIL_CLIENT';"

CLIENT_TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_CLIENT\", \"password\": \"$PASSWORD\"}" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: ${CLIENT_TOKEN:0:20}..."

# 3. Create Client Profile
echo -e "\n3. Creating Client Profile..."
curl -s -X POST "$API_URL/profile/client" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Test Client",
    "phone_number": "08011111111",
    "address": "123 Client St"
  }' | grep "message"

# 4. Register Driver
echo -e "\n4. Registering Driver..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_DRIVER\", \"password\": \"$PASSWORD\", \"role\": \"driver\"}" | grep "message"

# 5. Login Driver to get Token
echo -e "\n5. Logging in Driver..."
sqlite3 gopickup.db "UPDATE users SET is_verified=1 WHERE email='$EMAIL_DRIVER';"

DRIVER_TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_DRIVER\", \"password\": \"$PASSWORD\"}" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: ${DRIVER_TOKEN:0:20}..."

# 6. Create Driver Profile
echo -e "\n6. Creating Driver Profile..."
curl -s -X POST "$API_URL/profile/driver" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Test Driver",
    "phone_number": "08022222222",
    "license_number": "LIC-TEST-123",
    "vehicle_type": "Van",
    "plate_number": "PLT-TEST-123",
    "vehicle_capacity": 500.0
  }' | grep "message"

# 7. Create Admin (via DB hack since no register endpoint for admin)
echo -e "\n7. Creating Admin..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_ADMIN\", \"password\": \"$PASSWORD\", \"role\": \"client\"}" > /dev/null
# Promote to admin in DB and verify
sqlite3 gopickup.db "UPDATE users SET role='admin', is_verified=1 WHERE email='$EMAIL_ADMIN';"

# 8. Login Admin
echo -e "\n8. Logging in Admin..."
ADMIN_TOKEN=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL_ADMIN\", \"password\": \"$PASSWORD\"}" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "Token: ${ADMIN_TOKEN:0:20}..."

# 9. Get Driver ID (need to fetch it first)
# For this script, we'll cheat and query DB, or just use the /auth/me of driver if we stored it
# Let's query DB for simplicity in this shell script
DRIVER_ID=$(sqlite3 gopickup.db "SELECT id FROM users WHERE email='$EMAIL_DRIVER';")
echo "Driver ID: $DRIVER_ID"

# 10. Approve Driver
echo -e "\n10. Approving Driver..."
curl -s -X PATCH "$API_URL/admin/drivers/$DRIVER_ID/approve" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | grep "message"

# 11. Verify Approval (Check DB)
echo -e "\n11. Verifying Approval in DB..."
sqlite3 gopickup.db "SELECT is_approved FROM driver_profiles WHERE user_id='$DRIVER_ID';"

echo -e "\n=== Test Complete ==="
