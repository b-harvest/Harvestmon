package types

type Database struct {
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	DbName    string `yaml:"dbName"`
	AwsRegion string `yaml:"awsRegion"`
}

var (
	EnvDBName          = "DB_NAME"
	EnvDBAwsRegion     = "DB_AWS_REGION"
	EnvDBPort          = "DB_PORT"
	EnvDBHost          = "DB_HOST"
	EnvDBPassword      = "DB_PASSWORD"
	EnvDBUser          = "DB_USER"
	EnvMaxIdleConns    = "DB_MAX_IDLE_CONNS"
	EnvMaxOpenConns    = "DB_MAX_OPEN_CONNS"
	EnvConnMaxLifeTime = "DB_CONN_MAX_LIFE_TIME"
	EnvConnMaxIdleTime = "DB_CONN_MAX_IDLE_TIME"
)
