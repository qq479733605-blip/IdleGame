import axios from 'axios'

const http = axios.create({
    baseURL: 'http://localhost:8005',  // 修改为Gateway服务端口
    timeout: 5000,
})
export default http
