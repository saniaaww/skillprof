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

	JWTSecret           string
	JWTExpiresIn        int
	JWTRefreshExpiresIn int
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

		JWTSecret:           GetEnv("JWT_SECRET", ""),
		JWTExpiresIn:        GetEnvInt("JWT_EXPIRES_IN", 1),
		JWTRefreshExpiresIn: GetEnvInt("JWT_REFRESH_EXPIRES_IN", 168),
	}
}
