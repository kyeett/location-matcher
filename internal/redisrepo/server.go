package redisrepo

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	locationmodel "github.com/kyeett/location-matcher/internal/model"
)

type safeMap struct {
	m    map[string]time.Time
	lock sync.RWMutex
}

func (m *safeMap) get(key string) time.Time {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.m[key]
}

func (m *safeMap) set(key string, t time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m[key] = t
	fmt.Println("set", key, m.m[key])
}

type LocationServer struct {
	keyName string
	*redis.Pool
	positionLastSeen safeMap
}

func New(redisURL string, keyName string) (*LocationServer, error) {
	pool := &redis.Pool{
		MaxIdle:   10,
		MaxActive: 10,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redisURL)
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}
	return &LocationServer{
		Pool:    pool,
		keyName: keyName,
		positionLastSeen: safeMap{
			m:    map[string]time.Time{},
			lock: sync.RWMutex{},
		},
	}, nil
}

func (s *LocationServer) SetPosition(user string, pos locationmodel.Position) error {
	c := s.Pool.Get()
	defer c.Close()

	_, err := c.Do("GEOADD", s.keyName, pos.Longitude, pos.Latitude, "user:"+user)
	if err != nil {
		return err
	}

	s.positionLastSeen.set(user, time.Now())
	return nil
}

func (s *LocationServer) GetAllPositions() (map[string]locationmodel.Position, error) {
	c := s.Pool.Get()
	defer c.Close()

	values, err := redis.Values(c.Do("GEORADIUS", s.keyName, 0, 0, 22000, "km", "WITHCOORD"))
	if err != nil {
		return nil, err
	}

	positions := map[string]locationmodel.Position{}
	for _, v := range values {
		v2, _ := redis.Values(v, nil)

		var user string
		var pos locationmodel.Position
		_, err := redis.Scan(v2, &user, &pos)
		if err != nil {
			continue
		}

		user = strings.ReplaceAll(user, "user:", "")
		positions[user] = pos
	}
	return positions, nil
}

func (s *LocationServer) GetDistancesFrom(u string, maxDistanceKM float64) ([]locationmodel.Distance, error) {
	c := s.Pool.Get()
	defer c.Close()

	values, err := redis.Values(c.Do("GEORADIUSBYMEMBER", s.keyName, "user:"+u, maxDistanceKM, "m", "WITHDIST", "ASC"))
	if err != nil {
		return nil, err
	}

	distances := []locationmodel.Distance{}
	for _, v := range values {
		v2, _ := redis.Values(v, nil)

		var user string
		var dist float64
		_, err := redis.Scan(v2, &user, &dist)
		if err != nil {
			continue
		}

		user = strings.ReplaceAll(user, "user:", "")

		// Check if position has expired
		expired := s.positionLastSeen.get(user).Before(time.Now().Add(-600 * time.Second))

		distances = append(distances, locationmodel.Distance{
			User:     user,
			Distance: dist,
			Expired:  expired,
		})
	}
	return distances, nil
}
