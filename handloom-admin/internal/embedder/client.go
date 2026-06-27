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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

// apiGwV1Request matches the API Gateway REST API (v1) proxy payload format
// that httpadapter expects on the embedder Lambda. Embedder is now mounted
// behind the existing REST API rather than a Function URL.
//
// RequestContext.DomainName is populated so httpadapter can build a
// well-formed `http.Request` URL (`https://<domain>/embed`) rather than
// `https:///embed`; routing works either way but `r.Host` would otherwise
// be an empty string for any future middleware that inspects it.
type apiGwV1Request struct {
	HTTPMethod     string            `json:"httpMethod"`
	Path           string            `json:"path"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	IsBase64       bool              `json:"isBase64Encoded"`
	RequestContext apiGwV1RequestCtx `json:"requestContext"`
}
type apiGwV1RequestCtx struct {
	DomainName string `json:"domainName"`
	HTTPMethod string `json:"httpMethod"`
	Path       string `json:"path"`
}

type apiGwV1Response struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

// lambdaErrorPayload matches the JSON shape Lambda returns when the function
// crashes outside the handler (init failure, runtime exit, OOM) and
// `FunctionError` is NOT set on the SDK output. Treated as a separate path
// from a 200-with-empty-statusCode response so the caller gets a useful
// error message instead of `embedder returned 0:`.
type lambdaErrorPayload struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorType    string `json:"errorType"`
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

	outer := apiGwV1Request{
		HTTPMethod: "POST",
		Path:       "/embed",
		Headers: map[string]string{
			"Content-Type":         "application/json",
			"X-Embedder-Timestamp": ts,
			"X-Embedder-Nonce":     nonce,
			"X-Embedder-Signature": sig,
		},
		Body: string(body),
		RequestContext: apiGwV1RequestCtx{
			DomainName: "embedder.internal",
			HTTPMethod: "POST",
			Path:       "/embed",
		},
	}
	// Inject W3C traceparent into the synthetic APIGW headers so the embedder's
	// server span continues the caller's trace. This is a lambda.Invoke with a
	// hand-built payload — no transport-level header propagation happens otherwise.
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(outer.Headers))
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

	var resp apiGwV1Response
	if err := json.Unmarshal(out.Payload, &resp); err != nil {
		return nil, fmt.Errorf("decode embedder envelope: %w", err)
	}
	// Lambda runtime errors (OOM, init crash, panic) arrive as
	// {errorMessage, errorType} payloads with no statusCode field and no
	// FunctionError flag. Surface the real error instead of "returned 0:".
	if resp.StatusCode == 0 {
		var lerr lambdaErrorPayload
		if jerr := json.Unmarshal(out.Payload, &lerr); jerr == nil && lerr.ErrorMessage != "" {
			return nil, fmt.Errorf("embedder runtime error (%s): %s", lerr.ErrorType, lerr.ErrorMessage)
		}
		return nil, fmt.Errorf("embedder returned malformed envelope: %s", string(out.Payload))
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
