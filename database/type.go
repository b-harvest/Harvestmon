package types

type Database struct {
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	DbName    string `yaml:"dbName"`
	AwsRegion string `yaml:"awsRegion"`
}
