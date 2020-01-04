package redisrepo

import (
	"math"
	"testing"

	locationmodel "github.com/kyeett/location-matcher/internal/model"
	"github.com/stretchr/testify/assert"

	"github.com/alicebob/miniredis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const testKey = "testPos:"

func TestGetSetLocation(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	s, err := New("redis://"+mr.Addr(), testKey)
	// s, err := New("redis://localhost:6379")

	user := uuid.New().String()
	user2 := uuid.New().String()

	// Act
	pos := locationmodel.Position{-100, 40}
	err = s.SetPosition(user, pos)
	require.NoError(t, err)
	err = s.SetPosition(user2, locationmodel.Position{-100, 40})
	require.NoError(t, err)

	// Assert
	positions, err := s.GetAllPositions()
	require.NoError(t, err)

	_, exists := positions[user]
	assert.True(t, exists)
}

func TestGetDistance(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	s, err := New("redis://"+mr.Addr(), testKey)
	// s, err := New("redis://localhost:6379")

	requestingUser := uuid.New().String()
	otherUser1 := uuid.New().String()
	otherUser2 := uuid.New().String()

	// Act
	pos := locationmodel.Position{-100, 40}
	err = s.SetPosition(requestingUser, pos)
	require.NoError(t, err)

	// User 1
	err = s.SetPosition(otherUser1, locationmodel.Position{-100, 40.001})
	require.NoError(t, err)

	// User 2
	err = s.SetPosition(otherUser2, locationmodel.Position{-100, 40.004})
	require.NoError(t, err)

	// Assert
	distances, err := s.GetDistancesFrom(requestingUser, 100000)
	require.NoError(t, err)

	d1 := assertGetDistance(t, distances, otherUser1)
	d2 := assertGetDistance(t, distances, otherUser2)

	// 0.001 change in latitude is approximately 0.1114 kilometers
	change := 0.1114

	assert.True(t, isApproximately(d1.Distance, change))
	assert.True(t, isApproximately(d2.Distance, 4*change))
}

const tolerance = 0.001

func assertGetDistance(t *testing.T, distances []locationmodel.Distance, user string) locationmodel.Distance {
	for _, d := range distances {
		if d.User == user {
			return d
		}
	}
	t.Fatalf("failed to find distance for %q", user)
	return locationmodel.Distance{}
}

func isApproximately(d1, d2 float64) bool {
	return math.Abs(d1-d2) < tolerance
}
