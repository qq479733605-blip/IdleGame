// 简单的WebSocket连接测试
const WebSocket = require('ws');

const token = 'mock-jwt-testuser';
const wsUrl = `ws://localhost:8080/ws?token=${token}`;

console.log('正在连接到:', wsUrl);

const ws = new WebSocket(wsUrl);

ws.on('open', function open() {
    console.log('✅ WebSocket连接成功！');

    // 发送测试消息
    ws.send(JSON.stringify({ type: 'C_Login', token: token }));

    // 发送获取序列列表消息
    setTimeout(() => {
        ws.send(JSON.stringify({ type: 'C_ListSeq' }));
    }, 1000);

    // 5秒后发送停止消息
    setTimeout(() => {
        ws.send(JSON.stringify({ type: 'C_StopSeq' }));
    }, 5000);
});

ws.on('message', function message(data) {
    const msg = JSON.parse(data.toString());
    console.log('📨 收到消息:', msg);
});

ws.on('close', function close() {
    console.log('❌ WebSocket连接关闭');
});

ws.on('error', function error(err) {
    console.error('💥 WebSocket错误:', err);
});

// 10秒后自动关闭连接
setTimeout(() => {
    ws.close();
    process.exit(0);
}, 10000);