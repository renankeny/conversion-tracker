package config

import (
	"net"
	"net/url"
	"strings"
)

// ParseMinIOEndpoint normaliza endpoints S3/MinIO em formatos variados (ex: EasyPanel):
// aceita tanto "projeto_minio:9000" quanto "http://projeto_minio:9000" ou "https://s3.dominio.com/".
func ParseMinIOEndpoint(rawEndpoint, rawUseSSL string) (endpoint string, useSSL bool) {
	s := strings.TrimSpace(rawEndpoint)
	useSSL = strings.EqualFold(strings.TrimSpace(rawUseSSL), "true")
	if s == "" {
		return "", useSSL
	}

	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			if strings.EqualFold(u.Scheme, "https") {
				useSSL = true
			} else if strings.EqualFold(u.Scheme, "http") && strings.TrimSpace(rawUseSSL) == "" {
				useSSL = false
			}
			return u.Host, useSSL
		}
	}

	s = strings.TrimRight(s, "/")
	if idx := strings.IndexByte(s, '/'); idx > 0 {
		s = s[:idx]
	}
	return s, useSSL
}

// ApplyRedisURL preenche campos Redis a partir de REDIS_URL (padrão EasyPanel/Heroku:
// redis://default:senha@projeto_redis:6379/0 ou rediss://...).
func ApplyRedisURL(c *Config, rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || c == nil {
		return
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return
	}

	host := u.Host
	if _, _, splitErr := net.SplitHostPort(host); splitErr != nil {
		host = net.JoinHostPort(host, "6379")
	}
	c.RedisAddr = host

	if u.User != nil {
		if user := u.User.Username(); user != "" {
			c.RedisUsername = user
		}
		if pass, ok := u.User.Password(); ok {
			c.RedisPassword = pass
		}
	}
	if strings.EqualFold(u.Scheme, "rediss") {
		c.RedisTLS = true
	}
	if dbPath := strings.TrimPrefix(u.Path, "/"); dbPath != "" {
		c.RedisDB = atoi(dbPath)
	}
}

func (c *Config) MinIOConfigured() bool {
	return c != nil &&
		strings.TrimSpace(c.MinIOEndpoint) != "" &&
		strings.TrimSpace(c.MinIOAccessKey) != "" &&
		strings.TrimSpace(c.MinIOSecretKey) != "" &&
		strings.TrimSpace(c.MinIOBucket) != ""
}

