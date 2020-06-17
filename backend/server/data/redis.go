package data

import (
	"log"

	"github.com/go-redis/redis"
)

const prefix = "shorty"

// Redis data struct
type Redis struct {
	CharFloor int
	Client    *redis.Client
	Conn      *redis.Options
}

// Init connection to Redis server
func (r *Redis) Init() {
	r.Client = redis.NewClient(r.Conn)
	_, err := r.Client.Ping().Result()

	if err != nil {
		log.Fatal(err)
	}
}

// Set saves data to redis
func (r Redis) Set(url string) (string, error) {
	return "", nil
}

// Get loads data from redis
func (r Redis) Get(code string) (string, error) {
	return "", nil
}
