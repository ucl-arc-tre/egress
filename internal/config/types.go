package config

type S3CredentialBundle struct {
	Region          string
	AccessKeyId     string
	SecretAccessKey string
}

type AuthBasicCredentialsBundle struct {
	Username string
	Password string
}
