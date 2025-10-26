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
	// 处理预检请求
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
		return
	}

	// 调用下一个处理器
	c.handler.ServeHTTP(w, r)

	// 清除所有可能的CORS头，然后重新设置唯一的
	w.Header().Del("Access-Control-Allow-Origin")
	w.Header().Del("Access-Control-Allow-Methods")
	w.Header().Del("Access-Control-Allow-Headers")
	w.Header().Del("Access-Control-Max-Age")

	// 设置唯一的CORS头部
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
