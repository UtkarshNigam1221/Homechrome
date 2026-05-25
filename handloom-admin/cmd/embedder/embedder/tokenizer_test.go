package embedder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTokenizer_Encode uses the real tokenizer.json. Skipped in CI when
// the file is absent; downloaded via `make download-embedder-assets` locally.
func TestTokenizer_Encode(t *testing.T) {
	path := filepath.Join("..", "assets", "tokenizer.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("tokenizer.json missing at %s; run `make download-embedder-assets`", path)
	}

	tok, err := NewTokenizer(path, 128)
	require.NoError(t, err)
	defer tok.Close()

	ids, mask, err := tok.Encode("saree for wedding")
	require.NoError(t, err)
	require.Equal(t, 128, len(ids), "ids must be padded to max_len")
	require.Equal(t, 128, len(mask))
	require.NotZero(t, ids[0], "first token should be CLS / non-zero")
	nonZero := 0
	for _, id := range ids {
		if id != 0 {
			nonZero++
		}
	}
	require.GreaterOrEqual(t, nonZero, 3, "saree/for/wedding should produce >=3 non-zero tokens")
}
