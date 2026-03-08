package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	MinIO     MinIOConfig
	Upload    UploadConfig
	RateLimit RateLimitConfig
	Log       LogConfig
	Worker    WorkerConfig
	CORS      CORSConfig
}

type AppConfig struct {
	Name   string
	Env    string
	Port   int
	Secret string
}

func (a *AppConfig) IsProduction() bool { return a.Env == "production" }

func (a *AppConfig) IsDevelopment() bool { return a.Env == "development" }

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

func (r *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
	Region     string
}

type UploadConfig struct {
	MaxSize      int64
	ChunkSize    int64
	TempDir      string
	AllowedTypes []string
}

type RateLimitConfig struct {
	Max    int
	Expiry time.Duration
}

type LogConfig struct {
	Level  string
	Format string
	Output string
}

type WorkerConfig struct {
	Concurrency   int
	QueueDefault  string
	QueueCritical string
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

var (
	instance *Config
	once     sync.Once
)

// Load reads configuration from the .env file and environment variables,
// validates required fields, and returns the singleton Config instance.
// It panics on the first call if the configuration is invalid.
func Load() *Config {
	once.Do(func() {
		v := viper.New()

		v.SetConfigFile(".env")
		v.SetConfigType("env")
		if err := v.ReadInConfig(); err != nil {
			// In production the .env file is often absent; env vars are injected directly.
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: could not read .env file: %v\n", err)
			}
		}

		v.AutomaticEnv()
		setDefaults(v)
		bindEnvVars(v)

		cfg, err := buildConfig(v)
		if err != nil {
			panic(fmt.Sprintf("config: failed to build config: %v", err))
		}

		if err := validate(cfg); err != nil {
			panic(fmt.Sprintf("config: validation failed: %v", err))
		}

		instance = cfg
	})

	return instance
}

func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "file-management-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)

	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 25)
	v.SetDefault("database.max_lifetime", "5m")

	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	v.SetDefault("jwt.access_expiry", "15m")
	v.SetDefault("jwt.refresh_expiry", "168h") // 7 days

	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.bucket_name", "documents")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.region", "us-east-1")

	v.SetDefault("upload.max_size", int64(104857600))  // 100 MB
	v.SetDefault("upload.chunk_size", int64(5242880))  // 5 MB
	v.SetDefault("upload.temp_dir", "/tmp/uploads")
	v.SetDefault("upload.allowed_types",
		"application/pdf,image/jpeg,image/png,image/gif,image/webp,"+
			"application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,"+
			"text/plain")

	v.SetDefault("rate_limit.max", 100)
	v.SetDefault("rate_limit.expiry", "1m")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")

	v.SetDefault("worker.concurrency", 10)
	v.SetDefault("worker.queue_default", "default")
	v.SetDefault("worker.queue_critical", "critical")

	v.SetDefault("cors.allowed_origins", "*")
	v.SetDefault("cors.allowed_methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	v.SetDefault("cors.allowed_headers", "Origin,Content-Type,Accept,Authorization,X-Request-ID")
}

// bindEnvVars maps environment variable names to viper config keys so that
// plain ENV vars (e.g. DB_HOST) override nested viper keys (e.g. database.host).
func bindEnvVars(v *viper.Viper) {
	bindings := map[string]string{
		"app.name":   "APP_NAME",
		"app.env":    "APP_ENV",
		"app.port":   "APP_PORT",
		"app.secret": "APP_SECRET",

		"database.host":          "DB_HOST",
		"database.port":          "DB_PORT",
		"database.user":          "DB_USER",
		"database.password":      "DB_PASSWORD",
		"database.name":          "DB_NAME",
		"database.sslmode":       "DB_SSLMODE",
		"database.max_open_conns": "DB_MAX_OPEN_CONNS",
		"database.max_idle_conns": "DB_MAX_IDLE_CONNS",
		"database.max_lifetime":  "DB_MAX_LIFETIME",

		"redis.host":      "REDIS_HOST",
		"redis.port":      "REDIS_PORT",
		"redis.password":  "REDIS_PASSWORD",
		"redis.db":        "REDIS_DB",
		"redis.pool_size": "REDIS_POOL_SIZE",

		"jwt.access_secret":  "JWT_ACCESS_SECRET",
		"jwt.refresh_secret": "JWT_REFRESH_SECRET",
		"jwt.access_expiry":  "JWT_ACCESS_EXPIRY",
		"jwt.refresh_expiry": "JWT_REFRESH_EXPIRY",

		"minio.endpoint":    "MINIO_ENDPOINT",
		"minio.access_key":  "MINIO_ACCESS_KEY",
		"minio.secret_key":  "MINIO_SECRET_KEY",
		"minio.bucket_name": "MINIO_BUCKET_NAME",
		"minio.use_ssl":     "MINIO_USE_SSL",
		"minio.region":      "MINIO_REGION",

		"upload.max_size":      "UPLOAD_MAX_SIZE",
		"upload.chunk_size":    "UPLOAD_CHUNK_SIZE",
		"upload.temp_dir":      "UPLOAD_TEMP_DIR",
		"upload.allowed_types": "UPLOAD_ALLOWED_TYPES",

		"rate_limit.max":    "RATE_LIMIT_MAX",
		"rate_limit.expiry": "RATE_LIMIT_EXPIRY",

		"log.level":  "LOG_LEVEL",
		"log.format": "LOG_FORMAT",
		"log.output": "LOG_OUTPUT",

		"worker.concurrency":    "WORKER_CONCURRENCY",
		"worker.queue_default":  "WORKER_QUEUE_DEFAULT",
		"worker.queue_critical": "WORKER_QUEUE_CRITICAL",

		"cors.allowed_origins": "CORS_ALLOWED_ORIGINS",
		"cors.allowed_methods": "CORS_ALLOWED_METHODS",
		"cors.allowed_headers": "CORS_ALLOWED_HEADERS",
	}

	for key, env := range bindings {
		_ = v.BindEnv(key, env)
	}
}

func buildConfig(v *viper.Viper) (*Config, error) {
	cfg := &Config{}

	cfg.App = AppConfig{
		Name:   v.GetString("app.name"),
		Env:    v.GetString("app.env"),
		Port:   v.GetInt("app.port"),
		Secret: v.GetString("app.secret"),
	}

	cfg.Database = DatabaseConfig{
		Host:         v.GetString("database.host"),
		Port:         v.GetInt("database.port"),
		User:         v.GetString("database.user"),
		Password:     v.GetString("database.password"),
		Name:         v.GetString("database.name"),
		SSLMode:      v.GetString("database.sslmode"),
		MaxOpenConns: v.GetInt("database.max_open_conns"),
		MaxIdleConns: v.GetInt("database.max_idle_conns"),
		MaxLifetime:  parseDuration(v.GetString("database.max_lifetime"), 5*time.Minute),
	}

	cfg.Redis = RedisConfig{
		Host:     v.GetString("redis.host"),
		Port:     v.GetInt("redis.port"),
		Password: v.GetString("redis.password"),
		DB:       v.GetInt("redis.db"),
		PoolSize: v.GetInt("redis.pool_size"),
	}

	cfg.JWT = JWTConfig{
		AccessSecret:  v.GetString("jwt.access_secret"),
		RefreshSecret: v.GetString("jwt.refresh_secret"),
		AccessExpiry:  parseDuration(v.GetString("jwt.access_expiry"), 15*time.Minute),
		RefreshExpiry: parseDuration(v.GetString("jwt.refresh_expiry"), 7*24*time.Hour),
	}

	cfg.MinIO = MinIOConfig{
		Endpoint:   v.GetString("minio.endpoint"),
		AccessKey:  v.GetString("minio.access_key"),
		SecretKey:  v.GetString("minio.secret_key"),
		BucketName: v.GetString("minio.bucket_name"),
		UseSSL:     v.GetBool("minio.use_ssl"),
		Region:     v.GetString("minio.region"),
	}

	cfg.Upload = UploadConfig{
		MaxSize:      v.GetInt64("upload.max_size"),
		ChunkSize:    v.GetInt64("upload.chunk_size"),
		TempDir:      v.GetString("upload.temp_dir"),
		AllowedTypes: splitCSV(v.GetString("upload.allowed_types")),
	}

	cfg.RateLimit = RateLimitConfig{
		Max:    v.GetInt("rate_limit.max"),
		Expiry: parseDuration(v.GetString("rate_limit.expiry"), time.Minute),
	}

	cfg.Log = LogConfig{
		Level:  v.GetString("log.level"),
		Format: v.GetString("log.format"),
		Output: v.GetString("log.output"),
	}

	cfg.Worker = WorkerConfig{
		Concurrency:   v.GetInt("worker.concurrency"),
		QueueDefault:  v.GetString("worker.queue_default"),
		QueueCritical: v.GetString("worker.queue_critical"),
	}

	cfg.CORS = CORSConfig{
		AllowedOrigins: splitCSV(v.GetString("cors.allowed_origins")),
		AllowedMethods: splitCSV(v.GetString("cors.allowed_methods")),
		AllowedHeaders: splitCSV(v.GetString("cors.allowed_headers")),
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	type check struct {
		val  string
		name string
	}

	required := []check{
		{cfg.App.Secret, "APP_SECRET"},
		{cfg.Database.Host, "DB_HOST"},
		{cfg.Database.User, "DB_USER"},
		{cfg.Database.Password, "DB_PASSWORD"},
		{cfg.Database.Name, "DB_NAME"},
		{cfg.Redis.Host, "REDIS_HOST"},
		{cfg.JWT.AccessSecret, "JWT_ACCESS_SECRET"},
		{cfg.JWT.RefreshSecret, "JWT_REFRESH_SECRET"},
		{cfg.MinIO.AccessKey, "MINIO_ACCESS_KEY"},
		{cfg.MinIO.SecretKey, "MINIO_SECRET_KEY"},
	}

	var missing []string
	for _, c := range required {
		if strings.TrimSpace(c.val) == "" {
			missing = append(missing, c.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables:\n  - %s",
			strings.Join(missing, "\n  - "))
	}

	if cfg.App.Port <= 0 || cfg.App.Port > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535, got %d", cfg.App.Port)
	}

	if cfg.Upload.MaxSize <= 0 {
		return fmt.Errorf("UPLOAD_MAX_SIZE must be positive")
	}

	if cfg.Upload.ChunkSize <= 0 {
		return fmt.Errorf("UPLOAD_CHUNK_SIZE must be positive")
	}

	return nil
}

// parseDuration parses a duration string, handling the "Nd" (N days) shorthand
// that time.ParseDuration does not natively support.
func parseDuration(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}

	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
