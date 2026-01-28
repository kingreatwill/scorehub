package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr        string
	DBDSN       string
	TokenSecret string
	DevAuth     bool

	WeChatAppID  string
	WeChatSecret string

	TencentMapKey string
	AmapKey       string
}

func Load() Config {
	loadDotEnv()
	return Config{
		Addr:          getenv("SCOREHUB_ADDR", ":8080"),
		DBDSN:         getenv("SCOREHUB_DB_DSN", ""),
		TokenSecret:   getenv("SCOREHUB_TOKEN_SECRET", "change-me"),
		DevAuth:       getenvBool("SCOREHUB_DEV_AUTH", false),
		WeChatAppID:   getenv("SCOREHUB_WECHAT_APPID", ""),
		WeChatSecret:  getenv("SCOREHUB_WECHAT_SECRET", ""),
		TencentMapKey: getenv("SCOREHUB_TENCENT_MAP_KEY", ""),
		AmapKey:       getenv("SCOREHUB_AMAP_KEY", ""),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
