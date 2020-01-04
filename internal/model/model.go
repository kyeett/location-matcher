package locationmodel

import (
	"github.com/garyburd/redigo/redis"
)

type Position struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type Distance struct {
	User     string  `json:"user"`
	Distance float64 `json:"distance"`
}

func (p *Position) RedisScan(src interface{}) error {
	f, err := redis.Float64s(src, nil)
	if err != nil {
		return err
	}
	*p = Position{
		Longitude: f[0],
		Latitude:  f[1],
	}
	return nil
}
