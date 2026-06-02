package embedder

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/require"
)

// fakeInvoker captures the Invoke payload and returns a canned response.
type fakeInvoker struct {
	gotPayload []byte
	resp       []byte
}

func (f *fakeInvoker) Invoke(_ context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	f.gotPayload = in.Payload
	return &lambda.InvokeOutput{Payload: f.resp, StatusCode: 200}, nil
}

func TestClient_Embed_SerializesAndSigns(t *testing.T) {
	resp := `{"statusCode":200,"body":"{\"vectors\":[[0.1,0.2]],\"model\":\"x\"}"}`
	fi := &fakeInvoker{resp: []byte(resp)}
	c := NewClient(fi, "fnname", []byte("secret"), 5*time.Second)

	vecs, err := c.Embed(context.Background(), "hello")
	require.NoError(t, err)
	require.Equal(t, [][]float32{{0.1, 0.2}}, vecs)

	var outer map[string]any
	require.NoError(t, json.Unmarshal(fi.gotPayload, &outer))
	headers := outer["headers"].(map[string]any)
	require.NotEmpty(t, headers["X-Embedder-Signature"])
	require.NotEmpty(t, headers["X-Embedder-Timestamp"])
	require.NotEmpty(t, headers["X-Embedder-Nonce"])
}
