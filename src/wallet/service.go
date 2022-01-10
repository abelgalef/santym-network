package wallet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

type keyPair struct {
	rsa.PrivateKey
}

type Service interface {
	GenerateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey)
	ExportRsaPrivateKeyAsPemString(*rsa.PrivateKey) string
	ParseRsaPrivateKeyFromPemStr(string) (*rsa.PrivateKey, error)
	ExportRsaPublicKeyAsPemStr(*rsa.PublicKey) (string, error)
	ParseRsaPublicKeyFromPemStr(string) (*rsa.PublicKey, error)
}

type Wallet struct {
	Ballance        int
	PublicKeyString string
	PrivateKey      *rsa.PrivateKey
}

func NewWallet() *Wallet {
	privkey, pubkey := GenerateRsaKeyPair()
	pubkey_pem_str, err := ExportRsaPublicKeyAsPemStr(pubkey)
	if err != nil {
		panic(err)
	}

	return &Wallet{PublicKeyString: pubkey_pem_str, PrivateKey: privkey}
}

func GenerateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privkey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privkey, &privkey.PublicKey
}

func ExportRsaPrivateKeyAsPemString(privkey *rsa.PrivateKey) string {
	privkey_bytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkey_pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privkey_bytes,
	})
	return string(privkey_pem)
}

func ParseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse pem block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) (string, error) {
	pubkey_bytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", nil
	}

	pubkey_pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubkey_bytes,
	})

	return string(pubkey_pem), nil
}

func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break
	}

	return nil, errors.New("unsupported key type")
}
