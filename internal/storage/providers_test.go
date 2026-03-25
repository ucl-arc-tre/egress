package storage

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/storage/generic"
	"github.com/ucl-arc-tre/egress/internal/storage/s3"
	"github.com/ucl-arc-tre/egress/internal/types"
)

// Need to initialise config for this test
func TestS3StorageProvider(t *testing.T) {
	yaml := `
storage:
  provider: s3
  s3:
    region: us-east-1
    access_key_id: key1234
    secret_access_key: secret1234
dev:
  s3:
    url: ""
    bucket: ""
`
	cf := makeConfig(t, "s3.yaml", yaml)
	config.InitWithPath(cf)

	cfg := config.StorageConfigBundle{
		Provider: string(types.StorageProviderS3),
	}
	storage, err := Provider(cfg)
	assert.NoError(t, err)
	assert.IsType(t, &s3.Storage{}, storage)
}

// Need to initialise config for this test
func TestGenericStorageProvider(t *testing.T) {
	yaml := `
storage:
  provider: generic
  generic: {}
`
	cf := makeConfig(t, "generic.yaml", yaml)
	config.InitWithPath(cf)

	tlsDir := makeTLSDir(t)
	cfg := config.StorageConfigBundle{
		Provider:   string(types.StorageProviderGeneric),
		TLSCertDir: tlsDir,
	}
	storage, err := Provider(cfg)
	assert.NoError(t, err)
	assert.IsType(t, &generic.Storage{}, storage)
}

func TestUnsupportedProvider(t *testing.T) {
	cfg := config.StorageConfigBundle{
		Provider: "blah",
	}
	storage, err := Provider(cfg)
	assert.Error(t, err)
	assert.Nil(t, storage)
}

func makeConfig(t *testing.T, fileName string, yaml string) string {
	dir := t.TempDir()
	cf := filepath.Join(dir, fileName)

	err := os.WriteFile(cf, []byte(yaml), 0644)
	require.NoError(t, err, "Unable to create test config file")
	return cf
}

func makeTLSDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Generate CA key pair
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ca.crt"), caPEM, 0644))

	// Generate client key pair
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "egress-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	require.NoError(t, err)

	clientPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tls.crt"), clientPEM, 0644))

	clientKeyDER, err := x509.MarshalECPrivateKey(clientKey)
	require.NoError(t, err)

	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: clientKeyDER})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tls.key"), clientKeyPEM, 0600))

	return dir
}
