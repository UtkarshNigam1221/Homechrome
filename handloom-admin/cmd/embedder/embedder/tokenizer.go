package embedder

import (
	"fmt"
	"os"

	tk "github.com/daulet/tokenizers"
)

// Tokenizer wraps a HuggingFace tokenizer with fixed-length padding/truncation.
type Tokenizer struct {
	inner  *tk.Tokenizer
	maxLen int
}

// NewTokenizer loads a tokenizer.json from disk and configures it with the
// given maximum sequence length.
func NewTokenizer(path string, maxLen int) (*Tokenizer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer.json: %w", err)
	}
	inner, err := tk.FromBytes(b)
	if err != nil {
		return nil, fmt.Errorf("parse tokenizer.json: %w", err)
	}
	return &Tokenizer{inner: inner, maxLen: maxLen}, nil
}

// Close releases the underlying tokenizer resources.
func (t *Tokenizer) Close() { _ = t.inner.Close() }

// Encode tokenizes text, truncates to maxLen, and pads with zeros to exactly
// maxLen. Returns parallel int64 slices for input IDs and attention mask
// (1 for real tokens, 0 for padding).
func (t *Tokenizer) Encode(text string) ([]int64, []int64, error) {
	enc := t.inner.EncodeWithOptions(text, true,
		tk.WithReturnAttentionMask(),
	)

	rawIDs := enc.IDs
	rawMask := enc.AttentionMask

	// Truncate if longer than maxLen
	if len(rawIDs) > t.maxLen {
		rawIDs = rawIDs[:t.maxLen]
		rawMask = rawMask[:t.maxLen]
	}

	ids := make([]int64, t.maxLen)
	mask := make([]int64, t.maxLen)

	for i, id := range rawIDs {
		ids[i] = int64(id)
	}

	// Use attention mask from library if available; otherwise infer from token count
	if len(rawMask) > 0 {
		for i, m := range rawMask {
			mask[i] = int64(m)
		}
	} else {
		for i := range rawIDs {
			mask[i] = 1
		}
	}

	return ids, mask, nil
}
