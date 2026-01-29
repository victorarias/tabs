package server

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	AuthModeOff       = "off"
	AuthModeHeader    = "header"
	AuthModeIAPGoogle = "iap-google"
)

type AuthConfig struct {
	Mode          string
	HeaderUser    string
	IAPAudience   string
	IAPIssuer     string
	IAPJWKSURL    string
	IAPHTTPClient *http.Client
}

type Authenticator interface {
	Authenticate(r *http.Request) (string, error)
}

type NoAuth struct{}

func (NoAuth) Authenticate(_ *http.Request) (string, error) {
	return "", nil
}

type HeaderAuth struct {
	Header string
}

func (h HeaderAuth) Authenticate(r *http.Request) (string, error) {
	value := strings.TrimSpace(r.Header.Get(h.Header))
	if value == "" {
		return "", errors.New("authentication required")
	}
	return value, nil
}

type IAPAuth struct {
	audience string
	issuer   string
	jwksURL  string
	client   *http.Client
	cache    *jwksCache
}

func NewAuthenticator(cfg AuthConfig) (Authenticator, error) {
	mode := strings.TrimSpace(strings.ToLower(cfg.Mode))
	if mode == "" {
		mode = AuthModeOff
	}

	switch mode {
	case AuthModeOff:
		return NoAuth{}, nil
	case AuthModeHeader:
		header := strings.TrimSpace(cfg.HeaderUser)
		if header == "" {
			header = "X-Forwarded-User"
		}
		return HeaderAuth{Header: header}, nil
	case AuthModeIAPGoogle:
		if strings.TrimSpace(cfg.IAPAudience) == "" {
			return nil, errors.New("IAP_AUDIENCE is required for iap-google auth")
		}
		issuer := strings.TrimSpace(cfg.IAPIssuer)
		if issuer == "" {
			issuer = "https://cloud.google.com/iap"
		}
		jwksURL := strings.TrimSpace(cfg.IAPJWKSURL)
		if jwksURL == "" {
			jwksURL = "https://www.gstatic.com/iap/verify/public_key-jwk"
		}
		client := cfg.IAPHTTPClient
		if client == nil {
			client = &http.Client{Timeout: 5 * time.Second}
		}
		return &IAPAuth{
			audience: cfg.IAPAudience,
			issuer:   issuer,
			jwksURL:  jwksURL,
			client:   client,
			cache:    &jwksCache{keys: make(map[string]*rsa.PublicKey)},
		}, nil
	default:
		return nil, errors.New("unsupported AUTH_MODE")
	}
}

func (a *IAPAuth) Authenticate(r *http.Request) (string, error) {
	assertion := strings.TrimSpace(r.Header.Get("X-Goog-Iap-Jwt-Assertion"))
	if assertion == "" {
		return "", errors.New("IAP JWT assertion missing")
	}
	payload, err := a.verifyJWT(assertion)
	if err != nil {
		return "", err
	}
	user := strings.TrimSpace(payload.Email)
	if user == "" {
		user = strings.TrimSpace(payload.Sub)
	}
	if user == "" {
		return "", errors.New("IAP user missing")
	}
	return user, nil
}
