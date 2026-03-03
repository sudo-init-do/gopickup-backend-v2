#!/bin/bash
# set -e # Don't exit on error immediately, let us see the output

# Set environment variables
export APP_PORT=8080
export API_URL="http://localhost:$APP_PORT/api/v1"
export DB_NAME="gopickup.db"
TIMESTAMP=$(date +%s)

# Wait for server to be ready (assuming it's started separately)
echo "Waiting for server to be ready..."
until curl -s $API_URL/health > /dev/null; do
  sleep 1
done

echo "Server is ready."

# --- Helper Functions ---

register_user() {
  email=$1
  password=$2
  role=$3
  echo "Registering $role ($email)..."
  res=$(curl -s -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$email\", \"password\": \"$password\", \"role\": \"$role\"}")
  echo "Register Response: $res"

  # Auto-verify via SQLite
  echo "Auto-verifying $email via SQLite..."
  sqlite3 $DB_NAME "UPDATE users SET is_verified=1 WHERE email='$email';"
}

login_user() {
  email=$1
  password=$2
  echo "Logging in ($email)..." >&2
  response=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$email\", \"password\": \"$password\"}")
  echo "Login Response: $response" >&2
  token=$(echo $response | jq -r '.token')
  
  # Get User ID via /auth/me
  me_response=$(curl -s -X GET "$API_URL/auth/me" -H "Authorization: Bearer $token")
  user_id=$(echo $me_response | jq -r '.id')
  
  echo "$token:$user_id"
}

# --- 1. Vendor Setup ---
echo "--- 1. Vendor Setup ---"
VENDOR_EMAIL="vendor_${TIMESTAMP}@test.com"
register_user "$VENDOR_EMAIL" "password123" "vendor"
vendor_data=$(login_user "$VENDOR_EMAIL" "password123")
vendor_token=${vendor_data%%:*}
vendor_id=${vendor_data##*:}
echo "Vendor Token: $vendor_token"
echo "Vendor ID: $vendor_id"

echo "Creating Vendor Profile..."
res=$(curl -s -X POST "$API_URL/profile/vendor" \
  -H "Authorization: Bearer $vendor_token" \
  -H "Content-Type: application/json" \
  -d "{\"store_name\": \"Test Vendor $TIMESTAMP\", \"phone_number\": \"123${TIMESTAMP}\", \"business_type\": \"retail\", \"address\": \"123 Market St\"}")
echo "Create Profile Response: $res"

echo "Approving Vendor (Manual via SQLite)..."
sqlite3 $DB_NAME "UPDATE vendor_profiles SET is_approved=1 WHERE user_id='$vendor_id';"

echo "Creating Product..."
product_res=$(curl -s -X POST "$API_URL/vendor/products" \
  -H "Authorization: Bearer $vendor_token" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Product", "description": "Awesome product", "price": 100.00, "stock_quantity": 50, "category": "General"}')
echo "Create Product Response: $product_res"
product_id=$(echo $product_res | jq -r '.ID')
echo "Product created: $product_id"

# --- 2. Client Setup ---
echo "--- 2. Client Setup ---"
CLIENT_EMAIL="client_${TIMESTAMP}@test.com"
register_user "$CLIENT_EMAIL" "password123" "client"
client_data=$(login_user "$CLIENT_EMAIL" "password123")
client_token=${client_data%%:*}
client_id=${client_data##*:}

echo "Creating Client Profile..."
res=$(curl -s -X POST "$API_URL/profile/client" \
  -H "Authorization: Bearer $client_token" \
  -H "Content-Type: application/json" \
  -d "{\"full_name\": \"Test Client\", \"phone_number\": \"456${TIMESTAMP}\", \"address\": \"456 Home St\"}")
echo "Create Client Profile Response: $res"

echo "Creating Order (Checkout)..."
order_res=$(curl -s -X POST "$API_URL/orders/checkout" \
  -H "Authorization: Bearer $client_token" \
  -H "Content-Type: application/json" \
  -d "{\"items\": [{\"product_id\": \"$product_id\", \"quantity\": 1}], \"payment_method\": \"wallet\", \"delivery_address\": \"456 Home St\", \"pickup_address\": \"123 Market St\"}")
echo "Checkout Response: $order_res"
order_id=$(echo $order_res | jq -r '.ID')
echo "Order created: $order_id"

# --- 3. Update Order Status (Vendor) ---
echo "--- 3. Vendor Updates Order ---"
echo "Updating status to processing..."
curl -s -X PATCH "$API_URL/vendor/orders/$order_id/status" \
  -H "Authorization: Bearer $vendor_token" \
  -H "Content-Type: application/json" \
  -d '{"status": "processing"}'

echo "Updating status to searching_driver..."
curl -s -X PATCH "$API_URL/vendor/orders/$order_id/ready" \
  -H "Authorization: Bearer $vendor_token" \
  -H "Content-Type: application/json" \
  -d '{}'

# --- 4. Driver Setup ---
echo "--- 4. Driver Setup ---"
DRIVER_EMAIL="driver_${TIMESTAMP}@test.com"
register_user "$DRIVER_EMAIL" "password123" "driver"
driver_data=$(login_user "$DRIVER_EMAIL" "password123")
driver_token=${driver_data%%:*}
driver_id=${driver_data##*:}
echo "Driver ID: $driver_id"

echo "Creating Driver Profile..."
res=$(curl -s -X POST "$API_URL/profile/driver" \
  -H "Authorization: Bearer $driver_token" \
  -H "Content-Type: application/json" \
  -d "{\"full_name\": \"Test Driver\", \"phone_number\": \"789${TIMESTAMP}\", \"license_number\": \"LIC${TIMESTAMP}\", \"vehicle_type\": \"Van\", \"plate_number\": \"PLT${TIMESTAMP}\", \"vehicle_capacity\": 100}")
echo "Create Driver Profile Response: $res"

echo "Approving Driver (Manual via SQLite)... Driver ID: $driver_id"
sqlite3 $DB_NAME "UPDATE driver_profiles SET is_approved=1 WHERE user_id='$driver_id';"
sqlite3 $DB_NAME "UPDATE users SET is_verified=1 WHERE id='$driver_id';" # Just in case

echo "Driver Updating Location..."
curl -s -X PATCH "$API_URL/driver/location" \
  -H "Authorization: Bearer $driver_token" \
  -H "Content-Type: application/json" \
  -d '{"lat": 40.7128, "lng": -74.0060}'

echo "Driver Checking Available Jobs..."
jobs_res=$(curl -s -X GET "$API_URL/jobs/available" \
  -H "Authorization: Bearer $driver_token")
echo "Available Jobs: $jobs_res"

# --- 5. Bidding ---
echo "--- 5. Bidding ---"
echo "Driver Placing Bid..."
bid_res=$(curl -s -X POST "$API_URL/jobs/$order_id/bid" \
  -H "Authorization: Bearer $driver_token" \
  -H "Content-Type: application/json" \
  -d '{"amount": 15.00}')
echo "Bid Response: $bid_res"
bid_id=$(echo $bid_res | jq -r '.ID')
echo "Bid placed: $bid_id"

echo "Client Checking Bids..."
bids_res=$(curl -s -X GET "$API_URL/orders/$order_id/bids" \
  -H "Authorization: Bearer $client_token")
echo "Bids found: $bids_res"

echo "Client Accepting Bid..."
accept_res=$(curl -s -X POST "$API_URL/orders/$order_id/bids/$bid_id/accept" \
  -H "Authorization: Bearer $client_token")
echo "Bid accepted: $accept_res"

# --- 6. Final Verification ---
echo "--- 6. Final Verification ---"
order_final=$(curl -s -X GET "$API_URL/orders/$order_id" \
  -H "Authorization: Bearer $client_token")
status=$(echo $order_final | jq -r '.Status')
assigned_driver=$(echo $order_final | jq -r '.DriverID')

echo "Final Order Status: $status"
echo "Assigned Driver: $assigned_driver"

if [ "$status" == "assigned" ] && [ "$assigned_driver" == "$driver_id" ]; then
  echo "SUCCESS: Order assigned to driver correctly!"
else
  echo "FAILURE: Order status or driver assignment incorrect."
  exit 1
fi
