package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// GSI1SK is queried descending to get newest-first. Two links saved inside the same
// second must still order by time, not by their random id suffix — which is what a
// second-precision sort key did.
func TestUTMLink_SetKeys_SortsChronologicallyWithinASecond(t *testing.T) {
	base := time.Date(2026, 8, 25, 16, 56, 33, 0, time.UTC)

	// Ids chosen so a plain id comparison disagrees with creation order.
	earlier := &UTMLink{ID: "utm_f499fa26"}
	earlier.CreatedAt = base.Add(871 * time.Millisecond)
	earlier.SetKeys()

	later := &UTMLink{ID: "utm_77530564"}
	later.CreatedAt = base.Add(905 * time.Millisecond)
	later.SetKeys()

	assert.Less(t, earlier.GSI1SK, later.GSI1SK,
		"the link created later must sort after the earlier one")
	assert.Greater(t, earlier.ID, later.ID,
		"guard: this case is only meaningful while the ids sort opposite to the timestamps")
}

func TestUTMLink_SetKeys(t *testing.T) {
	link := &UTMLink{ID: "utm_abc123"}
	link.CreatedAt = time.Date(2026, 8, 25, 11, 26, 33, 871160000, time.UTC)
	link.SetKeys()

	assert.Equal(t, "UTM_LINK#utm_abc123", link.PK)
	assert.Equal(t, SKMetadata, link.SK)
	assert.Equal(t, "UTM_LINK#ALL", link.GSI1PK)
	assert.Equal(t, "2026-08-25T11:26:33.871160000Z#utm_abc123", link.GSI1SK)
	assert.Equal(t, "UTM_LINK", link.EntityType)
}
