package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Env         string
	HTTPAddr    string
	LogLevel    string
	JWTSecret   string
	JWTExpHours int

	AdminEmail    string
	AdminPassword string

	DatabaseURL string

	// Redis (asynq) — pra ambientes gerenciados (EasyPanel/Heroku/etc.) com auth/TLS.
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisUsername string
	RedisTLS      bool
	// RedisTLSInsecureSkipVerify=true desabilita validação de certificado.
	// Use só em ambientes com cert auto-assinado conhecido.
	RedisTLSInsecureSkipVerify bool

	GeoIPDBPath    string
	MetaAPIVersion string

	// MaxMindLicenseKey é usada pra baixar a base GeoLite2-City via endpoint
	// admin /admin/geoip/refresh. Pega em https://www.maxmind.com/en/geolite2/signup.
	MaxMindLicenseKey string

	// MetaPartnerAgent é recomendado pra agências/plataformas que enviam
	// eventos em nome de anunciantes. Formato: nome-versao (ex: "trackfind-1.0.0").
	// Vazio = não envia o campo (ok pra anunciantes diretos).
	MetaPartnerAgent string

	// PublicAPIURL é o fallback usado pelo /tracker.js gerado quando ele não
	// consegue detectar o próprio host. Em produção: "https://api.dominio.com/api/v1".
	PublicAPIURL string

	// AppURL é a URL pública do painel admin (Nuxt 4). Usada em redirects,
	// links em emails, CORS futuro do painel, etc.
	AppURL string

	// StrictCORS = true exige que o Origin de toda request casa com algum
	// domínio cadastrado em pixels.domains. Default false (dev).
	StrictCORS bool

	// Segredos opcionais usados na verificação de assinatura de webhooks.
	// Vazios desabilitam a checagem (mantemos o gateway funcionando sem
	// rejeitar payload — útil em dev).
	WebhookYampiSecret string

	// TrackingTargetHost é o host de destino que os clientes devem apontar
	// via CNAME quando configuram um tracking_host no pixel.
	// Ex: "track.trackfind.com.br". Em dev fica vazio (DNS não funciona em localhost).
	TrackingTargetHost string
}

func Load() (*Config, error) {
	c := &Config{
		Env:                getenv("APP_ENV", "development"),
		HTTPAddr:           getenv("HTTP_ADDR", ":8080"),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		JWTSecret:          getenv("JWT_SECRET", ""),
		JWTExpHours:        atoi(getenv("JWT_EXPIRES_HOURS", "24")),
		AdminEmail:         getenv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:      getenv("ADMIN_PASSWORD", ""),
		DatabaseURL:                getenv("DATABASE_URL", ""),
		RedisAddr:                  getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:              getenv("REDIS_PASSWORD", ""),
		RedisUsername:              getenv("REDIS_USERNAME", ""),
		RedisDB:                    atoi(getenv("REDIS_DB", "0")),
		RedisTLS:                   getenv("REDIS_TLS", "false") == "true",
		RedisTLSInsecureSkipVerify: getenv("REDIS_TLS_INSECURE_SKIP_VERIFY", "false") == "true",
		GeoIPDBPath:        getenv("GEOIP_DB_PATH", ""),
		MetaAPIVersion:     getenv("META_API_VERSION", "v25.0"),
		MaxMindLicenseKey:  getenv("MAXMIND_LICENSE_KEY", ""),
		MetaPartnerAgent:   getenv("META_PARTNER_AGENT", ""),
		PublicAPIURL:       getenv("PUBLIC_API_URL", "http://localhost:8080/api/v1"),
		AppURL:             getenv("APP_URL", "http://localhost:3000"),
		StrictCORS:         getenv("STRICT_CORS", "false") == "true",
		WebhookYampiSecret: getenv("WEBHOOK_YAMPI_SECRET", ""),
		TrackingTargetHost: getenv("TRACKING_TARGET_HOST", ""),
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
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
