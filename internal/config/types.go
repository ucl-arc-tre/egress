package config

type S3CredentialBundle struct {
	Region          string
	AccessKeyId     string
	SecretAccessKey string
}

type DBConfigBundle struct {
	Provider       string
	BaseURL        string
	Username       string
	Password       string
	ReadinessProbe string
}

type AuthBasicCredentialsBundle struct {
	Username string
	Password string
}
