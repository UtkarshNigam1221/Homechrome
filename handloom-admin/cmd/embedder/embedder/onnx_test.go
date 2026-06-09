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
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("model file missing: %s", modelPath)
	}

	// onnxruntime is dlopen'd at libPath, so it must be a runtime built for the
	// HOST OS/arch. The bundled assets/libonnxruntime.so targets the Lambda
	// (linux/arm64) and won't load on a dev mac, so this inference test is
	// opt-in: set ONNXRUNTIME_SHARED_LIB_PATH to a host-native runtime to run
	// it. Skip otherwise (also covers CI, which has no embedder assets).
	libPath := os.Getenv("ONNXRUNTIME_SHARED_LIB_PATH")
	if libPath == "" {
		t.Skip("ONNXRUNTIME_SHARED_LIB_PATH unset; skipping ONNX inference test (needs a host-native runtime)")
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
