package common

import (
	"net/http"
)

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	handler http.Handler
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(handler http.Handler) *CORSMiddleware {
	return &CORSMiddleware{
		handler: handler,
	}
}

func (c *CORSMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 首先设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")

	// 处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 调用下一个处理器
	c.handler.ServeHTTP(w, r)
}
