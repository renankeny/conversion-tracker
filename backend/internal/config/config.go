package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type Config struct {
	Env         string
	HTTPAddr    string
	LogLevel    string
	JWTSecret   string
	JWTExpHours int

	AdminEmail    string
	AdminPassword string

	DatabaseURL       string
	DBMaxConns        int32
	WorkerConcurrency int

	// Redis (Asynq, cache e Pub/Sub) — suporta REDIS_URL (EasyPanel) ou REDIS_ADDR.
	RedisAddr                  string
	RedisPassword              string
	RedisDB                    int
	RedisUsername              string
	RedisTLS                   bool
	RedisTLSInsecureSkipVerify bool

	GeoIPDBPath       string
	MetaAPIVersion    string
	MaxMindLicenseKey string

	SentEventsRetentionDays    int
	EventCountersRetentionDays int
	WhatsAppMediaRetentionDays int

	MetaPartnerAgent  string
	TikTokPartnerName string

	// Google Ads
	GoogleAdsDeveloperToken string
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	GoogleOAuthRedirectURI  string

	// MinIO / S3 Storage (mídias WhatsApp com retenção automática)
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIORegion    string
	MinIOUseSSL    bool

	TraefikHTTPSecret  string
	EncryptionKey      string
	PublicAPIURL       string
	AppURL             string
	StrictCORS         bool
	TrackingTargetHost string

	// System Monitor SSH Credentials
	Node1IP   string
	Node1User string
	Node1Pass string
	Node2IP   string
	Node2User string
	Node2Pass string
}

func Load() (*Config, error) {
	minioEndpoint, minioSSL := ParseMinIOEndpoint(
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_USE_SSL"),
	)
	mediaRet := atoi(getenv("WHATSAPP_MEDIA_RETENTION_DAYS", "14"))
	if mediaRet <= 0 {
		mediaRet = 14
	}

	c := &Config{
		Env:                        getenv("APP_ENV", "development"),
		HTTPAddr:                   getenv("HTTP_ADDR", ":8080"),
		LogLevel:                   getenv("LOG_LEVEL", "info"),
		JWTSecret:                  getenv("JWT_SECRET", ""),
		JWTExpHours:                atoi(getenv("JWT_EXPIRES_HOURS", "24")),
		AdminEmail:                 getenv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:              getenv("ADMIN_PASSWORD", ""),
		DatabaseURL:                getenv("DATABASE_URL", ""),
		DBMaxConns:                 int32(atoi(getenv("DB_MAX_CONNS", "20"))),
		WorkerConcurrency:          atoi(getenv("WORKER_CONCURRENCY", "50")),
		RedisAddr:                  getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:              getenv("REDIS_PASSWORD", ""),
		RedisUsername:              getenv("REDIS_USERNAME", ""),
		RedisDB:                    atoi(getenv("REDIS_DB", "0")),
		RedisTLS:                   getenv("REDIS_TLS", "false") == "true",
		RedisTLSInsecureSkipVerify: getenv("REDIS_TLS_INSECURE_SKIP_VERIFY", "false") == "true",
		GeoIPDBPath:                getenv("GEOIP_DB_PATH", ""),
		MetaAPIVersion:             getenv("META_API_VERSION", "v26.0"),
		MaxMindLicenseKey:          getenv("MAXMIND_LICENSE_KEY", ""),
		SentEventsRetentionDays:    atoi(getenv("SENT_EVENTS_RETENTION_DAYS", "7")),
		EventCountersRetentionDays: atoi(getenv("EVENT_COUNTERS_RETENTION_DAYS", "90")),
		WhatsAppMediaRetentionDays: mediaRet,
		MetaPartnerAgent:           getenv("META_PARTNER_AGENT", ""),
		TikTokPartnerName:          getenv("TIKTOK_PARTNER_NAME", ""),

		GoogleAdsDeveloperToken: getenv("GOOGLE_ADS_DEVELOPER_TOKEN", ""),
		GoogleOAuthClientID:     getenv("GOOGLE_OAUTH_CLIENT_ID", ""),
		GoogleOAuthClientSecret: getenv("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		GoogleOAuthRedirectURI:  getenv("GOOGLE_OAUTH_REDIRECT_URI", ""),

		MinIOEndpoint:  minioEndpoint,
		MinIOAccessKey: getenv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey: getenv("MINIO_SECRET_KEY", ""),
		MinIOBucket:    getenv("MINIO_BUCKET", "trackeado-whatsapp"),
		MinIORegion:    getenv("MINIO_REGION", ""),
		MinIOUseSSL:    minioSSL,

		TraefikHTTPSecret:  getenv("TRAEFIK_HTTP_SECRET", ""),
		EncryptionKey:      getenv("ENCRYPTION_KEY", ""),
		PublicAPIURL:       getenv("PUBLIC_API_URL", "http://localhost:8080/api/v1"),
		AppURL:             getenv("APP_URL", "http://localhost:3000"),
		StrictCORS:         getenv("STRICT_CORS", "false") == "true",
		TrackingTargetHost: getenv("TRACKING_TARGET_HOST", ""),

		Node1IP:   getenv("NODE1_IP", ""),
		Node1User: getenv("NODE1_USER", "root"),
		Node1Pass: getenv("NODE1_PASS", ""),
		Node2IP:   getenv("NODE2_IP", ""),
		Node2User: getenv("NODE2_USER", "root"),
		Node2Pass: getenv("NODE2_PASS", ""),
	}

	ApplyRedisURL(c, os.Getenv("REDIS_URL"))

	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.EncryptionKey) != 32 {
		c.EncryptionKey = util.DeriveEncryptionKey(c.JWTSecret)
	}
	return c, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
