package verifier

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// decryptRequest/decryptResponse mirror docs/extension-contract.md section 3
// exactly: base64 wire encoding (tee-node is Go and marshals []byte as
// base64 in JSON) -- explicitly flagged there as "the single most common
// porting mistake" versus the hex encoding used elsewhere in the wire
// protocol.
type decryptRequest struct {
	EncryptedMessage string `json:"encryptedMessage"`
}

type decryptResponse struct {
	DecryptedMessage string `json:"decryptedMessage"`
}

// NodeDecrypter calls tee-node's local POST /decrypt on SIGN_PORT --
// "never exposed outside the container", so this must only ever be pointed
// at localhost.
type NodeDecrypter struct {
	baseURL string
	client  *http.Client
}

func NewNodeDecrypter(signPort int, client *http.Client) *NodeDecrypter {
	if client == nil {
		client = http.DefaultClient
	}
	return &NodeDecrypter{
		baseURL: fmt.Sprintf("http://localhost:%d", signPort),
		client:  client,
	}
}

// Decrypt decrypts a ciphertext that was encrypted to the TEE's public key.
func (d *NodeDecrypter) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	reqBody, err := json.Marshal(decryptRequest{
		EncryptedMessage: base64.StdEncoding.EncodeToString(ciphertext),
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling decrypt request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+"/decrypt", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("building decrypt request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling local node /decrypt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading decrypt response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// Never log body contents here: on an error path the local node
		// could echo back something derived from the plaintext bundle, and
		// SPEC.md section 11 requires plaintext never enter a log.
		return nil, fmt.Errorf("local node /decrypt returned status %d", resp.StatusCode)
	}

	var decoded decryptResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decoding decrypt response: %w", err)
	}

	plaintext, err := base64.StdEncoding.DecodeString(decoded.DecryptedMessage)
	if err != nil {
		return nil, fmt.Errorf("base64-decoding decrypted message: %w", err)
	}
	return plaintext, nil
}
