package queue

import (
	"crypto/tls"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
)

// BuildRedisClientOpt monta a configuração do cliente asynq/redis a partir
// do config carregado do .env. Suporta auth (username/password), DB index
// e TLS (com opção de skip-verify pra ambientes com cert auto-assinado).
//
// Tudo é opcional: a config mínima é só `RedisAddr`.
func BuildRedisClientOpt(cfg *config.Config) asynq.RedisClientOpt {
	opt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	if cfg.RedisTLS {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.RedisTLSInsecureSkipVerify, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
		}
	}
	return opt
}

// BuildRedisClient cria um *redis.Client direto (sem asynq) usando as mesmas
// configurações de BuildRedisClientOpt. Usado pelo worker para publicar
// eventos no canal Redis pub/sub do hub WebSocket.
func BuildRedisClient(cfg *config.Config) *redis.Client {
	opt := &redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	if cfg.RedisTLS {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.RedisTLSInsecureSkipVerify, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
		}
	}
	return redis.NewClient(opt)
}
