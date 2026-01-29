package server

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

type iapClaims struct {
	Iss   string      `json:"iss"`
	Aud   interface{} `json:"aud"`
	Sub   string      `json:"sub"`
	Email string      `json:"email"`
	Exp   int64       `json:"exp"`
	Iat   int64       `json:"iat"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type jwksCache struct {
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func (a *IAPAuth) verifyJWT(token string) (*iapClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return nil, errors.New("invalid JWT header")
	}
	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return nil, errors.New("invalid JWT payload")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid JWT signature")
	}

	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, errors.New("invalid JWT header")
	}
	if header.Alg != "RS256" {
		return nil, errors.New("unsupported JWT algorithm")
	}

	var claims iapClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, errors.New("invalid JWT payload")
	}

	if claims.Iss != a.issuer {
		return nil, errors.New("invalid JWT issuer")
	}
	if !audienceMatches(claims.Aud, a.audience) {
		return nil, errors.New("invalid JWT audience")
	}
	if claims.Exp == 0 || time.Now().Unix() > claims.Exp {
		return nil, errors.New("JWT expired")
	}

	key, err := a.getKey(header.Kid)
	if err != nil {
		return nil, err
	}

	signed := parts[0] + "." + parts[1]
	hash := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sig); err != nil {
		return nil, errors.New("JWT signature invalid")
	}

	return &claims, nil
}

func decodeSegment(seg string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(seg)
}

func audienceMatches(aud interface{}, expected string) bool {
	switch v := aud.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}

func (a *IAPAuth) getKey(kid string) (*rsa.PublicKey, error) {
	if kid == "" {
		return nil, errors.New("JWT missing kid")
	}

	a.cache.mu.Lock()
	key := a.cache.keys[kid]
	a.cache.mu.Unlock()
	if key != nil {
		return key, nil
	}

	if err := a.refreshKeys(); err != nil {
		return nil, err
	}

	a.cache.mu.Lock()
	key = a.cache.keys[kid]
	a.cache.mu.Unlock()
	if key == nil {
		return nil, errors.New("JWT key not found")
	}
	return key, nil
}

func (a *IAPAuth) refreshKeys() error {
	resp, err := a.client.Get(a.jwksURL)
	if err != nil {
		return errors.New("failed to fetch IAP JWKS")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to fetch IAP JWKS: %s", resp.Status)
	}

	var payload struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return errors.New("failed to parse IAP JWKS")
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range payload.Keys {
		if key.Kty != "RSA" || key.Kid == "" {
			continue
		}
		pub, err := buildRSAKey(key.N, key.E)
		if err != nil {
			continue
		}
		keys[key.Kid] = pub
	}

	if len(keys) == 0 {
		return errors.New("no valid IAP JWKS keys")
	}

	a.cache.mu.Lock()
	a.cache.keys = keys
	a.cache.fetchedAt = time.Now()
	a.cache.mu.Unlock()

	return nil
}

func buildRSAKey(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	modulus := new(big.Int).SetBytes(nBytes)
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: modulus, E: exponent}, nil
}
