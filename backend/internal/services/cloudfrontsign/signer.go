package cloudfrontsign

import (
	"crypto"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
)

type Signer struct {
	urlSigner *sign.URLSigner
}

func New(domain, keyPairID, privateKeyPEM string) (*Signer, error) {
	if domain == "" || keyPairID == "" || privateKeyPEM == "" {
		return nil, errors.New("cloudfrontsign: domain, keyPairID and privateKeyPEM are required")
	}
	pem := strings.ReplaceAll(privateKeyPEM, `\n`, "\n")

	// Try PKCS#1 (BEGIN RSA PRIVATE KEY) first, then PKCS#8 (BEGIN PRIVATE KEY)
	var signer crypto.Signer
	if k, err := sign.LoadPEMPrivKey(strings.NewReader(pem)); err == nil {
		signer = k
	} else if s, err2 := sign.LoadPEMPrivKeyPKCS8AsSigner(strings.NewReader(pem)); err2 == nil {
		signer = s
	} else {
		return nil, errors.New("cloudfrontsign: failed to parse private key (tried PKCS#1 and PKCS#8)")
	}

	return &Signer{urlSigner: sign.NewURLSigner(keyPairID, signer)}, nil
}

// SignURL signs rawURL with a CloudFront Canned Policy expiring at expires.
func (s *Signer) SignURL(rawURL string, expires time.Time) (string, error) {
	return s.urlSigner.Sign(rawURL, expires)
}
