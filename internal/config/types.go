package config

type StorageConfigBundle struct {
	Provider string
	S3       S3StorageConfig
}

type S3StorageConfig struct {
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
	Password string // #nosec G117 -- read only from k8s Secret
}

type AuthBasicCredentialsBundle struct {
	Username string
	Password string // #nosec G117 -- read only from k8s Secret
}
