const WebSocket = require('ws');

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjE2NTg0MzYsImlhdCI6MTc2MTU3MjAzNiwicGxheWVySUQiOiI3ZGMxM2E5Y2NhZWI0MmVmIn0.bGSYrcgk73SIN4wlHqu5rZcgXI3U4LNzplSts2F1_4o";
const ws = new WebSocket(`ws://localhost:8002/ws?token=${token}`);

ws.on('open', function open() {
    console.log('[WS] Connected to gateway');

    // Send login message
    ws.send(JSON.stringify({ type: 'C_Login', token: token }));
    console.log('[WS] Sent login message');
});

ws.on('message', function message(data) {
    try {
        const msg = JSON.parse(data);
        console.log('[WS] Received:', msg);

        if (msg.type === 'S_LoginOK') {
            console.log('[WS] ✅ Login successful! PlayerID:', msg.playerId);

            // Test ping/pong
            setTimeout(() => {
                console.log('[WS] Sending ping test...');
                ws.send(JSON.stringify({ type: 'C_Ping' }));
            }, 1000);

        } else if (msg.type === 'S_Pong') {
            console.log('[WS] ✅ Ping/Pong working!');

            // Close connection after successful test
            setTimeout(() => {
                console.log('[WS] ✅ Test completed - closing connection');
                ws.close();
            }, 1000);
        } else {
            console.log('[WS] Received other message:', msg);
        }
    } catch (err) {
        console.error('[WS] Failed to parse message:', err);
        console.log('[WS] Raw data:', data.toString());
    }
});

ws.on('close', function close(code, reason) {
    console.log(`[WS] Connection closed: ${code} - ${reason}`);
});

ws.on('error', function error(err) {
    console.error('[WS] Error:', err);
});

// Test timeout after 30 seconds
setTimeout(() => {
    if (ws.readyState === WebSocket.OPEN) {
        console.log('[WS] Test timeout - closing connection');
        ws.close();
    }
}, 30000);