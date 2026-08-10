package verifier

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestNodeDecrypterUsesBase64WireFormat(t *testing.T) {
	// docs/extension-contract.md section 3: base64, not hex -- "the single
	// most common porting mistake". Assert the request body is genuinely
	// base64, and that a real base64-encoded response round-trips.
	plaintext := []byte(`{"formatVersion":1}`)
	var capturedRequestBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/decrypt" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedRequestBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		resp := decryptResponse{DecryptedMessage: base64.StdEncoding.EncodeToString(plaintext)}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	port := mustPort(t, srv.URL)
	d := NewNodeDecrypter(port, srv.Client())

	got, err := d.Decrypt(context.Background(), []byte("ciphertext-bytes"))
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("Decrypt returned %q, want %q", got, plaintext)
	}

	sentEncrypted, ok := capturedRequestBody["encryptedMessage"]
	if !ok {
		t.Fatal("request body missing encryptedMessage field")
	}
	if _, err := base64.StdEncoding.DecodeString(sentEncrypted); err != nil {
		t.Fatalf("encryptedMessage was not valid base64: %v (this is the porting mistake the doc warns about)", err)
	}
	decodedBack, _ := base64.StdEncoding.DecodeString(sentEncrypted)
	if string(decodedBack) != "ciphertext-bytes" {
		t.Fatalf("round-trip mismatch: got %q", decodedBack)
	}
}

func TestNodeDecrypterErrorsOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewNodeDecrypter(mustPort(t, srv.URL), srv.Client())
	if _, err := d.Decrypt(context.Background(), []byte("x")); err == nil {
		t.Fatal("expected an error on a non-200 response")
	}
}

func mustPort(t *testing.T, rawURL string) int {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}
	port, err := strconv.Atoi(strings.Split(u.Host, ":")[1])
	if err != nil {
		t.Fatalf("parsing test server port: %v", err)
	}
	return port
}
