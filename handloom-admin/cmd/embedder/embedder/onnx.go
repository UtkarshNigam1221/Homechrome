package embedder

import (
	"fmt"
	"math"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// ortOnce ensures the ORT environment is initialized exactly once per process.
var (
	ortOnce    sync.Once
	ortInitErr error
)

// ONNXSession wraps a loaded ONNX model session plus the tokenizer.
// It is safe to call Embed from multiple goroutines (mutex-protected).
//
// Lambda concurrency note (C4): Lambda's runtime API is strictly
// request/response — each container processes one request at a time, so
// mu cannot actually contend in Lambda.  The mutex is defense-in-depth for
// local-dev paths (make run-embedder-local) where multiple goroutines may
// share a single ONNXSession concurrently.  No concurrency cap is needed at
// the Lambda level; AWS naturally isolates concurrent requests into separate
// containers, each with its own session.
type ONNXSession struct {
	sess   *ort.DynamicAdvancedSession
	tok    *Tokenizer
	maxLen int
	mu     sync.Mutex
}

// inputNames are the standard names exported by optimum-cli for XLM-R / indic-sbert.
var (
	onnxInputNames  = []string{"input_ids", "attention_mask", "token_type_ids"}
	onnxOutputNames = []string{"last_hidden_state"}
)

// NewONNXSession loads the ONNX model at modelPath, initializes the ORT
// environment (once), and wraps the tokenizer at tokenizerPath.
//
//   - If libPath is non-empty it is passed to SetSharedLibraryPath before
//     InitializeEnvironment (must be called before the first session is created).
//   - maxLen controls the fixed sequence length used for tokenization.
func NewONNXSession(modelPath, libPath, tokenizerPath string, maxLen int) (*ONNXSession, error) {
	// Initialize the ORT runtime environment exactly once.
	ortOnce.Do(func() {
		if libPath != "" {
			ort.SetSharedLibraryPath(libPath)
		}
		ortInitErr = ort.InitializeEnvironment()
	})
	if ortInitErr != nil {
		return nil, fmt.Errorf("onnxruntime init: %w", ortInitErr)
	}

	tok, err := NewTokenizer(tokenizerPath, maxLen)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	sess, err := ort.NewDynamicAdvancedSession(modelPath, onnxInputNames, onnxOutputNames, nil)
	if err != nil {
		tok.Close()
		return nil, fmt.Errorf("create onnx session: %w", err)
	}

	return &ONNXSession{
		sess:   sess,
		tok:    tok,
		maxLen: maxLen,
	}, nil
}

// Close releases the ONNX session and tokenizer resources.
func (s *ONNXSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess != nil {
		_ = s.sess.Destroy()
		s.sess = nil
	}
	if s.tok != nil {
		s.tok.Close()
		s.tok = nil
	}
}

// Embed tokenizes each text, runs the ONNX model, mean-pools the token
// embeddings (over non-padding positions), and L2-normalizes the result.
// Returns one []float32 of length EmbeddingDim per input text.
func (s *ONNXSession) Embed(texts []string) ([][]float32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := s.embedOne(text)
		if err != nil {
			return nil, fmt.Errorf("embed[%d] %q: %w", i, text, err)
		}
		results[i] = vec
	}
	return results, nil
}

// embedOne runs a single inference. Called with s.mu held.
func (s *ONNXSession) embedOne(text string) ([]float32, error) {
	ids, mask, err := s.tok.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}

	seqLen := int64(s.maxLen)
	shape := ort.NewShape(1, seqLen)

	// token_type_ids — all zeros (XLM-R doesn't use segment ids but the export expects them)
	typeIDs := make([]int64, s.maxLen)

	idsTensor, err := ort.NewTensor(shape, ids)
	if err != nil {
		return nil, fmt.Errorf("create input_ids tensor: %w", err)
	}
	defer idsTensor.Destroy() //nolint:errcheck

	maskTensor, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("create attention_mask tensor: %w", err)
	}
	defer maskTensor.Destroy() //nolint:errcheck

	typeIDsTensor, err := ort.NewTensor(shape, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("create token_type_ids tensor: %w", err)
	}
	defer typeIDsTensor.Destroy() //nolint:errcheck

	// Output tensor: [1, maxLen, EmbeddingDim]
	outShape := ort.NewShape(1, seqLen, int64(EmbeddingDim))
	outData := make([]float32, s.maxLen*EmbeddingDim)
	outTensor, err := ort.NewTensor(outShape, outData)
	if err != nil {
		return nil, fmt.Errorf("create output tensor: %w", err)
	}
	defer outTensor.Destroy() //nolint:errcheck

	inputs := []ort.Value{idsTensor, maskTensor, typeIDsTensor}
	outputs := []ort.Value{outTensor}

	if err := s.sess.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("onnx run: %w", err)
	}

	hidden := outTensor.GetData()
	return meanPoolNormalize(hidden, mask, s.maxLen, EmbeddingDim), nil
}

// meanPoolNormalize mean-pools hidden states over non-padding token positions
// (mask[i] == 1) and L2-normalizes the result.
func meanPoolNormalize(hidden []float32, mask []int64, seq, dim int) []float32 {
	v := make([]float32, dim)
	var n float32
	for i := 0; i < seq; i++ {
		if mask[i] == 0 {
			continue
		}
		n++
		base := i * dim
		for j := 0; j < dim; j++ {
			v[j] += hidden[base+j]
		}
	}
	if n > 0 {
		for j := range v {
			v[j] /= n
		}
	}
	var mag float32
	for _, x := range v {
		mag += x * x
	}
	if mag > 0 {
		inv := float32(1.0 / math.Sqrt(float64(mag)))
		for j := range v {
			v[j] *= inv
		}
	}
	return v
}
