package config

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Key Derivation Function = argon2i
// Version =  0x13 (19)
// Memory = 512
// Time = 2
// Threads = 2
// Salt = ILoveSaltCakes!!!
//
// Adheres to PHC string format
// https://github.com/P-H-C/phc-string-format/blob/master/phc-sf-spec.md
const argon2PasswordDigest = `$argon2i$v=19$m=512,t=2,p=2$SUxvdmVTYWx0Q2FrZXMhISE$UgSWnBB5OkdqMAu+OfvwNLVMUijMnnmVm0kRSfmS9E8`
const argon2Password = `Foobar`
const username = `wfauser`

func TestBasicAuth(t *testing.T) {
	cfg := &Config{
		Auth: &Auth{
			Username:     username,
			PasswordHash: argon2PasswordDigest,
		},
	}
	authMiddleware := BasicAuth(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "User authenticated successfully!")
	})
	t.Run("test basic auth", func(t *testing.T) {
		t.Parallel()
		t.Run("success cases", func(t *testing.T) {
			ts := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					authMiddleware(w, r, next)
				}))
			defer ts.Close()
			req, err := http.NewRequest("GET", ts.URL, nil)
			if err != nil {
				log.Fatal(err)
			}
			req.SetBasicAuth(username, argon2Password)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Expected no error, instead got: '%v'", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected no error, instead got: '%v'", resp.Status)
			}
		})
		t.Run("failure cases", func(t *testing.T) {
			ts := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					authMiddleware(w, r, next)
				}))
			defer ts.Close()
			req, err := http.NewRequest("GET", ts.URL, nil)
			if err != nil {
				log.Fatal(err)
			}
			passwordDigestTooShort := `$argon2i$v=19$m=512,t=2,p=2$SUxvdmVTYWx0Q2FrZXMhISE$UgSWnBB5OkdqMAu+Ofvw9E8`
			req.SetBasicAuth("incorrectUser", passwordDigestTooShort)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Expected no error, instead got: '%v'", err)
			}
			if resp.StatusCode == http.StatusOK {
				t.Errorf("Expected an error, instead got: '%v'", resp.Status)
			}
		})
	})
}
func TestArgon2KeyDerivationFunction(t *testing.T) {
	a := &argon2Kdf{}
	digest, salt, err := a.ParsePHCString(argon2PasswordDigest)
	if err != nil {
		t.Errorf("Expected no error, instead got: '%v'", err)
	}
	t.Run("parse PHC String", func(t *testing.T) {
		t.Parallel()
		t.Run("success cases", func(t *testing.T) {
			wantDigest, err := base64.RawStdEncoding.DecodeString("UgSWnBB5OkdqMAu+OfvwNLVMUijMnnmVm0kRSfmS9E8")
			if err != nil {
				t.Errorf("Expected no error, instead got: '%v'", err)
			}
			if subtle.ConstantTimeCompare(digest, wantDigest) != 1 {
				t.Errorf("Expected: '%s', got: '%s'", wantDigest, digest)
			}
			if subtle.ConstantTimeCompare(salt, []byte("ILoveSaltCakes!!!")) != 1 {
				t.Errorf("Expected: '%s', got: '%s'", "ILoveSaltCakes!!!", salt)
			}
		})
		t.Run("failure cases", func(t *testing.T) {
			wantDigest, err := base64.RawStdEncoding.DecodeString("UgSWnBB5OkdqMAu+sfvwNLVMUijMnnmVm0kRSfmS9")
			if err != nil {
				_, ok := err.(base64.CorruptInputError)
				if !ok {
					t.Error("Got an unexpected error")
				}
			}
			if subtle.ConstantTimeCompare(digest, wantDigest) == 1 {
				t.Error("Expected error")
			}
			if subtle.ConstantTimeCompare(salt, []byte("ILSaltCakes!!!")) == 1 {
				t.Error("Expected error")
			}
		})

	})
	t.Run("generate digest", func(t *testing.T) {
		t.Parallel()
		t.Run("success cases", func(t *testing.T) {
			got, err := a.Key([]byte(argon2Password), salt)
			if err != nil {
				t.Errorf("Expected no error, instead got: '%v'", err)
			}
			want, _ := base64.RawStdEncoding.DecodeString("UgSWnBB5OkdqMAu+OfvwNLVMUijMnnmVm0kRSfmS9E8")
			if subtle.ConstantTimeCompare(got, want) != 1 {
				t.Errorf("Expected: '%x', got: '%x'", want, got)
			}
		})
		t.Run("failure cases", func(t *testing.T) {
			// will call a.Key with nil parameters
			a = nil
			got, err := a.Key([]byte(argon2Password), salt)
			if err == nil {
				t.Errorf("Expected error: %s", got)
			}
			want, _ := base64.RawStdEncoding.DecodeString("UgSWnBB5OkdqMAu+OfvwNLVMUijMnnmVm0kRSfmS9E8")
			if subtle.ConstantTimeCompare(got, want) == 1 {
				t.Errorf("Expected: '%x', got: '%x'", want, got)
			}
		})
	})
}
