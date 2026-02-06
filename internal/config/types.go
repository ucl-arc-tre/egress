package config

type S3CredentialBundle struct {
	Region          string
	AccessKeyId     string
	SecretAccessKey string
}

type DBConfigBundle struct {
	Provider string
	Rqlite   RqliteConfig
}

type RqliteConfig struct {
	BaseURL  string
	Username string
	Password string
}

type AuthBasicCredentialsBundle struct {
	Username string
	Password string
}
