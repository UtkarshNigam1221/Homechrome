package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucketForDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{1 * time.Second, "le_30s"},
		{30 * time.Second, "le_30s"},
		{31 * time.Second, "le_2m"},
		{2 * time.Minute, "le_2m"},
		{10 * time.Minute, "le_10m"},
		{1 * time.Hour, "le_1h"},
		{2 * time.Hour, "le_inf"},
	}
	for _, c := range cases {
		got := BucketForDuration(c.in, DurationCartToPaymentBoundaries, DurationCartToPaymentLabels)
		assert.Equal(t, c.want, got, "duration %v", c.in)
	}
}

func TestBucketForCartSize(t *testing.T) {
	assert.Equal(t, "1", BucketForCartSize(0))
	assert.Equal(t, "1", BucketForCartSize(1))
	assert.Equal(t, "2_3", BucketForCartSize(2))
	assert.Equal(t, "2_3", BucketForCartSize(3))
	assert.Equal(t, "4_5", BucketForCartSize(4))
	assert.Equal(t, "4_5", BucketForCartSize(5))
	assert.Equal(t, "6_10", BucketForCartSize(10))
	assert.Equal(t, "11_plus", BucketForCartSize(11))
	assert.Equal(t, "11_plus", BucketForCartSize(100))
}
