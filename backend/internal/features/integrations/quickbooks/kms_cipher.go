package quickbooks

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

const kmsCiphertextPrefix = "kms:v1:"

type kmsAPI interface {
	Encrypt(context.Context, *kms.EncryptInput, ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(context.Context, *kms.DecryptInput, ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

// KMSCipher keeps OAuth tokens out of application-managed encryption keys.
type KMSCipher struct {
	client kmsAPI
	keyID  string
}

func NewKMSCipher(ctx context.Context, keyID, region string) (*KMSCipher, error) {
	if strings.TrimSpace(keyID) == "" || strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("AWS KMS key ID and region are required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return &KMSCipher{client: kms.NewFromConfig(cfg), keyID: keyID}, nil
}

func (c *KMSCipher) Encrypt(plaintext string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := c.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId: &c.keyID, Plaintext: []byte(plaintext),
		EncryptionContext: map[string]string{"application": "ledgerops", "purpose": "quickbooks-oauth"},
	})
	if err != nil {
		return nil, fmt.Errorf("encrypt QuickBooks token with AWS KMS: %w", err)
	}
	encoded := kmsCiphertextPrefix + base64.RawStdEncoding.EncodeToString(output.CiphertextBlob)
	return []byte(encoded), nil
}

func (c *KMSCipher) Decrypt(ciphertext []byte) (string, error) {
	value := string(ciphertext)
	if !strings.HasPrefix(value, kmsCiphertextPrefix) {
		return "", fmt.Errorf("QuickBooks token was not encrypted by AWS KMS")
	}
	blob, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, kmsCiphertextPrefix))
	if err != nil {
		return "", fmt.Errorf("decode AWS KMS ciphertext: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := c.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:    blob,
		EncryptionContext: map[string]string{"application": "ledgerops", "purpose": "quickbooks-oauth"},
	})
	if err != nil {
		return "", fmt.Errorf("decrypt QuickBooks token with AWS KMS: %w", err)
	}
	return string(output.Plaintext), nil
}
