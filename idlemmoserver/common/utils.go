package common

import (
	"encoding/json"
)

// Unmarshal 通用的反序列化函数
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Marshal 通用的序列化函数
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
