package config

type AppConfig struct {
	AppName    string
	AppEnv     string
	AppPort    int
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

func LoadAppConfig() AppConfig {
	return AppConfig{
		AppName:    GetEnv("APP_NAME", "SkillProf"),
		AppEnv:     GetEnv("APP_ENV", "development"),
		AppPort:    GetEnvInt("APP_PORT", 3000),
		DBHost:     GetEnv("DB_HOST", "localhost"),
		DBPort:     GetEnvInt("DB_PORT", 5432),
		DBUser:     GetEnv("DB_USER", "postgres"),
		DBPassword: GetEnv("DB_PASSWORD", ""),
		DBName:     GetEnv("DB_NAME", "skillprof"),
		DBSSLMode:  GetEnv("DB_SSL_MODE", "disable"),
	}
}
