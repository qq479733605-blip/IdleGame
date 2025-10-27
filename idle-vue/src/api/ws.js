import mitt from 'mitt'
import { useUserStore } from '../store/user.js'

const emitter = mitt()
let ws = null
let heartbeatTimer = null

export function connectWS(token) {
    console.log('Connecting WebSocket with token:', token ? 'present' : 'MISSING');

    if (!token) {
        console.error('No token provided for WebSocket connection');
        return;
    }

    const url = `ws://localhost:8005/ws?token=${token}`  // 修改为Gateway服务端口
    ws = new WebSocket(url)

    ws.onopen = () => {
        console.log('[WS] connected')
        send({ type: 'C_Login', token })

        const player = useUserStore()
        console.log('WS connected')

        clearInterval(heartbeatTimer)
        heartbeatTimer = setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
                send({ type: 'C_Ping' })
            }
        }, 25000)
    }

    ws.onmessage = (ev) => {
        const msg = JSON.parse(ev.data)

        // 处理心跳响应
        if (msg.type === 'S_Pong') {
            console.log('[WS] received pong')
            return
        }

        emitter.emit('message', msg)
    }

    ws.onclose = () => {
        clearInterval(heartbeatTimer)
        console.warn('[WS] closed, retrying...')
        const player = useUserStore()
        console.log('WS closed, retrying...')
        setTimeout(() => connectWS(token), 2000)
    }
}

export function send(data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(data))
    }
}

export default emitter
