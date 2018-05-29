package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"golang.org/x/crypto/argon2"
)

var ErrUnauthorized = errors.New("error: unable to authorize user")
var RealmMessage = `Authentication required for API access to workflow db connector`

type keyDerivationFn interface {
	Key(password, salt []byte) ([]byte, error)
	ParsePHCString(PHCStringHash string) (digest, salt []byte, err error)
}

type argon2Kdf struct {
	name    string
	version string
	memory  uint32
	time    uint32
	threads uint8
}

// BasicAuth reads the stored username and password info from the config file
// and returns a negroni middleware implementing HTTP Basic Authentication
func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		storedUsername, storedPasswordHash := getStoredUsernamePassword(config.Options)
		kdf, err := selectKdf(storedPasswordHash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		storedDigest, storedSalt, err := kdf.ParsePHCString(storedPasswordHash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		digest, err := kdf.Key([]byte(password), storedSalt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !ok || subtle.ConstantTimeCompare([]byte(username), []byte(storedUsername)) != 1 ||
			subtle.ConstantTimeCompare(digest, storedDigest) != 1 {
			w.Header().Set("WWW-Authenticate", "Basic realm="+RealmMessage)
			http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func selectKdf(hash string) (keyDerivationFn, error) {
	switch {
	case strings.HasPrefix(hash, "$argon2"):
		return &argon2Kdf{}, nil
	default:
		return nil, ErrUnauthorized
	}
}

func (a *argon2Kdf) ParsePHCString(PHCStringHash string) (digest, salt []byte, err error) {
	if a == nil {
		return nil, nil, errors.New("Please initialize a new argon2Kdf first")
	}
	validArgon2Hash := regexp.MustCompile(`\$([^\$]+)\$v=([0-9]+)\$m=([0-9]+),t=([0-9]+),p=([0-9]+)\$([^\$]+)\$([^\$]+)`)
	matches := validArgon2Hash.FindStringSubmatch(PHCStringHash)
	memory, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, nil, err
	}
	time, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, nil, err
	}
	threads, err := strconv.Atoi(matches[5])
	if err != nil {
		return nil, nil, err
	}
	a.name = matches[1]
	a.version = matches[2]
	a.memory = uint32(memory)
	a.time = uint32(time)
	a.threads = uint8(threads)
	salt, err = base64.RawStdEncoding.DecodeString(matches[6])
	if err != nil {
		return nil, nil, err
	}
	digest, err = base64.RawStdEncoding.DecodeString(matches[7])
	if err != nil {
		return nil, nil, err
	}
	return digest, salt, err
}
func (a *argon2Kdf) Key(password, salt []byte) ([]byte, error) {
	if a == nil || password == nil || salt == nil {
		return nil, errors.New("error: argon2Kdf should be initialized and password and salt should be non-nil")
	}
	// default to digest length of 32 bytes
	digest := argon2.Key(password, salt, a.time, a.memory, a.threads, 32)
	if len(digest) != 32 {
		return nil, errors.New("error: can not generate digest")
	}
	return digest, nil
}

func getStoredUsernamePassword(cfg config.Config) (username, passwordHash string) {
	return cfg.Auth.Username, cfg.Auth.PasswordHash
}
