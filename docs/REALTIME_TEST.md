# Realtime Verification Guide (WebSocket)

This document explains how to verify the WebSocket functionality for real-time updates in GoPickup.

## 1. Connection Details

- **Endpoint**: `wss://your-staging-domain.com/api/v1/ws` (or `ws://localhost:8080/api/v1/ws` locally)
- **Method**: GET
- **Authentication**: JWT Token required in `Authorization` header.

**Example Header:**
```
Authorization: Bearer <YOUR_JWT_TOKEN>
```

## 2. Testing with Postman or wscat

### Using `wscat` (CLI)
Since standard `wscat` might not support headers easily depending on version, you can try:
```bash
wscat -c ws://localhost:8080/api/v1/ws -H "Authorization: Bearer $TOKEN"
```

### Using Postman
1.  Open Postman -> New -> WebSocket Request.
2.  Enter URL: `ws://localhost:8080/api/v1/ws`
3.  Go to **Headers** tab.
4.  Key: `Authorization`, Value: `Bearer <YOUR_TOKEN>`
5.  Click **Connect**.
6.  You should see "Connected" status.

## 3. Event Verification Flow

Open two connections:
1.  **Client** (Login as Client, Connect WS)
2.  **Driver** (Login as Driver, Connect WS)

### Scenario A: New Message
1.  **Client**: Initiate a chat via REST API (`POST /api/v1/chats/initiate`).
2.  **Client**: Send a message via REST API (`POST /api/v1/chats/:id/messages`).
3.  **Driver**: Watch the WebSocket stream.
4.  **Expectation**: Driver receives a JSON message:
    ```json
    {
      "type": "new_message",
      "payload": {
        "chat_id": "...",
        "content": "Hello",
        "sender_id": "..."
      }
    }
    ```

### Scenario B: Order Status Update
1.  **Client**: Create an order.
2.  **Vendor**: Mark order as `READY` via REST API.
3.  **Client**: Watch WebSocket stream.
4.  **Expectation**: Client receives status update event.

### Scenario C: Bid Received
1.  **Driver**: Place a bid on an order via REST API.
2.  **Client**: Watch WebSocket stream.
3.  **Expectation**: Client receives a "new_bid" notification.

## 4. Troubleshooting

- **Connection Failed (401)**: Token is invalid or expired. Check `Authorization` header.
- **Connection Failed (403)**: Token valid but user blocked or not verified.
- **No Messages**: Ensure you are listening to the right events. The backend sends events based on User ID targeting.
- **Heartbeat**: The server might send PING/PONG frames. Ensure your client handles them.
