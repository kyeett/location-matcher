package redisrepo

import (
	"strings"

	"github.com/garyburd/redigo/redis"
	locationmodel "github.com/kyeett/location-matcher/internal/model"
)

type LocationServer struct {
	keyName string
	*redis.Pool
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
	}, nil
}

func (s *LocationServer) SetPosition(user string, pos locationmodel.Position) error {
	c := s.Pool.Get()
	defer c.Close()

	_, err := c.Do("GEOADD", s.keyName, pos.Longitude, pos.Latitude, "user:"+user)
	if err != nil {
		return err
	}
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

func (s *LocationServer) GetDistancesFrom(user string, maxDistanceKM float64) ([]locationmodel.Distance, error) {
	c := s.Pool.Get()
	defer c.Close()

	values, err := redis.Values(c.Do("GEORADIUSBYMEMBER", s.keyName, "user:"+user, maxDistanceKM, "m", "WITHDIST", "ASC"))
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
		distances = append(distances, locationmodel.Distance{
			User:     user,
			Distance: dist,
		})
	}
	return distances, nil
}
