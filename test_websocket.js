const WebSocket = require('ws');

const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjE2NTc3MjgsImlhdCI6MTc2MTU3MTMyOCwicGxheWVySUQiOiJlZmRlNDMxMTBlNjE0OThiIn0.8AgVA88mlbYvz3ccGZ4zZ3nId8GqWysYVphoqM-_5Kk";
const ws = new WebSocket(`ws://localhost:8002/ws?token=${token}`);

let pingCount = 0;
let pongCount = 0;
let loginReceived = false;

ws.on('open', function open() {
    console.log('[WS] Connected to gateway');

    // Send login message
    ws.send(JSON.stringify({ type: 'C_Login', token: token }));
    console.log('[WS] Sent login message');

    // Start sending ping every 15 seconds
    setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
            pingCount++;
            console.log(`[WS] Sending ping #${pingCount}`);
            ws.send(JSON.stringify({ type: 'C_Ping' }));
        }
    }, 15000);
});

ws.on('message', function message(data) {
    try {
        const msg = JSON.parse(data);
        console.log('[WS] Received:', msg);

        if (msg.type === 'S_LoginOK') {
            loginReceived = true;
            console.log('[WS] Login successful! PlayerID:', msg.playerId);
        } else if (msg.type === 'S_Pong') {
            pongCount++;
            console.log(`[WS] Received pong #${pongCount}`);
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
    console.log(`[WS] Stats: ${pingCount} pings sent, ${pongCount} pongs received, login: ${loginReceived}`);
});

ws.on('error', function error(err) {
    console.error('[WS] Error:', err);
});

// Test timeout after 2 minutes
setTimeout(() => {
    if (ws.readyState === WebSocket.OPEN) {
        console.log('[WS] Test completed - closing connection');
        ws.close();
    }
}, 120000);