package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtPrivateKey *rsa.PrivateKey
	jwtPublicKey  *rsa.PublicKey
)

// Load or generate RSA keys for JWT
func InitJWTKeys(privPath, pubPath string) error {
	var err error
	if _, err = os.Stat(privPath); os.IsNotExist(err) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		privPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
		os.WriteFile(privPath, privPem, 0600)
		pubBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
		os.WriteFile(pubPath, pubPem, 0644)
	}
	privPem, err := os.ReadFile(privPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(privPem)
	if block == nil {
		return errors.New("invalid private key PEM")
	}
	jwtPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	pubPem, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	block, _ = pem.Decode(pubPem)
	if block == nil {
		return errors.New("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	var ok bool
	jwtPublicKey, ok = pub.(*rsa.PublicKey)
	if !ok {
		return errors.New("not RSA public key")
	}
	return nil
}

// Generate a JWT for a user
func GenerateJWT(userID, userType string, scopes []string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"type":   userType,
		"scopes": scopes,
		"exp":    time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(jwtPrivateKey)
}

// Parse and validate a JWT, returns claims if valid
func ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtPublicKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
