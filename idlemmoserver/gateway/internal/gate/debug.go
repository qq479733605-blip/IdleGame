package gate

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

// handleDebug 调试端点 - 打印请求详情
func (s *Service) handleDebug(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== DEBUG REQUEST ===")
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Headers:")
	for name, values := range r.Header {
		log.Printf("  %s: %v", name, values)
	}
	log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))

	// 读取请求体
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
		} else {
			log.Printf("Body: %s", string(body))
			// 重新设置请求体供后续使用
			r.Body = io.NopCloser(strings.NewReader(string(body)))
		}
	}
	log.Printf("===================")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Debug info logged",
		"method":  r.Method,
		"url":     r.URL.String(),
	})
}
