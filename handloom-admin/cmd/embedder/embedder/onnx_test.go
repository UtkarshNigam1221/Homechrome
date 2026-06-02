package embedder

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestONNX_EmbedReturnsUnitVectors(t *testing.T) {
	modelPath := filepath.Join("..", "assets", "model-int8.onnx")
	tokPath := filepath.Join("..", "assets", "tokenizer.json")
	libPath := os.Getenv("ONNXRUNTIME_SHARED_LIB_PATH")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("model file missing: %s", modelPath)
	}

	sess, err := NewONNXSession(modelPath, libPath, tokPath, 128)
	require.NoError(t, err)
	defer sess.Close()

	vecs, err := sess.Embed([]string{"saree", "kanjivaram silk"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)
	require.Equal(t, EmbeddingDim, len(vecs[0]))

	// L2-normalized → magnitude ≈ 1.0
	var sumSq float32
	for _, x := range vecs[0] {
		sumSq += x * x
	}
	mag := math.Sqrt(float64(sumSq))
	require.InDelta(t, 1.0, mag, 0.01, "vector must be L2-normalized")
}
