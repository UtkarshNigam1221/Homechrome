package store

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func TestToStoreProduct_IncludesVideo(t *testing.T) {
	p := &domain.Product{
		ID:             "p1",
		VideoURL:       "https://cdn.example.com/assets/VIDEO/x.mp4",
		VideoPosterURL: "https://cdn.example.com/assets/IMAGE/x.jpg",
	}
	sp := toStoreProduct(p)
	require.Equal(t, p.VideoURL, sp.VideoURL)
	require.Equal(t, p.VideoPosterURL, sp.VideoPosterURL)
}
