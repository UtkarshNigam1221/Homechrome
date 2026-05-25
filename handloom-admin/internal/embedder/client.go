// Package embedder provides a small client used by other Lambdas to call the
// embedder Lambda's /embed endpoint via the AWS Lambda SDK (IAM auth) while
// also signing the request body with HMAC for the embedder's server-side
// verification.
package embedder

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// Invoker is the subset of lambda.Client used here; abstracted for tests.
type Invoker interface {
	Invoke(ctx context.Context, in *lambda.InvokeInput, opts ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// Client wraps the embedder Lambda.
type Client struct {
	inv     Invoker
	fnName  string
	authKey []byte
	timeout time.Duration
}

// NewClient constructs a Client. timeout is applied per Embed call; if zero,
// it defaults to 10s. Note: authKey is sensitive; callers should fetch it from
// SSM SecureString and hold it only in-process.
func NewClient(inv Invoker, fnName string, authKey []byte, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{inv: inv, fnName: fnName, authKey: authKey, timeout: timeout}
}

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Vectors [][]float32 `json:"vectors"`
	Model   string      `json:"model"`
}

// fnURLRequest matches the API Gateway v2 HTTP payload format that Lambda
// Function URLs deliver. The fields we set are sufficient for chi-proxy to
// reconstruct an http.Request.
type fnURLRequest struct {
	Version        string            `json:"version"`
	RawPath        string            `json:"rawPath"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	IsBase64       bool              `json:"isBase64Encoded"`
	RequestContext fnURLRequestCtx   `json:"requestContext"`
}
type fnURLRequestCtx struct {
	HTTP fnURLHTTP `json:"http"`
}
type fnURLHTTP struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type fnURLResponse struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

// Embed sends `texts` to the embedder and returns the resulting vectors.
func (c *Client) Embed(ctx context.Context, texts ...string) ([][]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	body, err := json.Marshal(embedRequest{Texts: texts})
	if err != nil {
		return nil, err
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	sig := sign(c.authKey, ts, nonce, body)

	outer := fnURLRequest{
		Version: "2.0",
		RawPath: "/embed",
		Headers: map[string]string{
			"Content-Type":         "application/json",
			"X-Embedder-Timestamp": ts,
			"X-Embedder-Nonce":     nonce,
			"X-Embedder-Signature": sig,
		},
		Body: string(body),
		RequestContext: fnURLRequestCtx{
			HTTP: fnURLHTTP{Method: "POST", Path: "/embed"},
		},
	}
	payload, err := json.Marshal(outer)
	if err != nil {
		return nil, err
	}

	out, err := c.inv.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(c.fnName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke embedder: %w", err)
	}
	if out.FunctionError != nil && *out.FunctionError != "" {
		return nil, fmt.Errorf("embedder lambda error (%s): %s", aws.ToString(out.FunctionError), string(out.Payload))
	}

	var resp fnURLResponse
	if err := json.Unmarshal(out.Payload, &resp); err != nil {
		return nil, fmt.Errorf("decode embedder envelope: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embedder returned %d: %s", resp.StatusCode, resp.Body)
	}
	var er embedResponse
	if err := json.Unmarshal([]byte(resp.Body), &er); err != nil {
		return nil, fmt.Errorf("decode embedder body: %w", err)
	}
	return er.Vectors, nil
}

func sign(key []byte, ts, nonce string, body []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(ts))
	h.Write([]byte("\n"))
	h.Write([]byte(nonce))
	h.Write([]byte("\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
