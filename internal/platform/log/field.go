package log

import "go.uber.org/zap"

type Field = zap.Field

func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}
func String(key string, val string) Field {
	return zap.String(key, val)
}

func Err(err error) Field {
	return zap.Error(err)
}

func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}
