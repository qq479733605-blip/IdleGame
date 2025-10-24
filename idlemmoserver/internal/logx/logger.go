package logx

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	// 控制台友好输出
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(output)
}

func Info(msg string, fields ...interface{}) {
	log.Info().Fields(mapify(fields...)).Msg(msg)
}

func Warn(msg string, fields ...interface{}) {
	log.Warn().Fields(mapify(fields...)).Msg(msg)
}

func Error(msg string, fields ...interface{}) {
	log.Error().Fields(mapify(fields...)).Msg(msg)
}

func mapify(fields ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}
	for i := 0; i < len(fields)-1; i += 2 {
		k, ok := fields[i].(string)
		if !ok {
			continue
		}
		m[k] = fields[i+1]
	}
	return m
}
