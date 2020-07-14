package data

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
	"github.com/hashicorp/go-hclog"
)

var log = hclog.Default()

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
		log.Error("Error", "Server Ping Failed", err)
	} else {
		log.Info("Redis Server", "Connection", "Online")
	}

}

// Save saves data to redis
func (r *Redis) Save(url string) (string, error) {
	// discover returns a uniqe code doesnt exist
	code, err := generate(r.Client, r.CharFloor)
	if err != nil {
		return "", err
	}
	err = set(r.Client, url, code)
	if err != nil {
		return "", err
	}
	log.Info("Redis Save", "Code stored", hclog.Fmt("Code: %s URL: %s", code, url))
	return code, nil
}

// Load grabs data from redis
func (r Redis) Load(code string) (string, error) {
	fullURL, err := r.Client.Do("get", code).String()

	if err == redis.Nil {
		return "", errors.New("Code not found")
	} else if err != nil {
		log.Error("Redis Load", "Error", err)
		return "", err
	}

	log.Info("Redis Load", "URL retrieved", hclog.Fmt("%s", fullURL))
	return fullURL, nil
}

// set inserts the code:url as key:value
func set(c *redis.Client, code string, fullURL string) error {
	/* Sets the code as the key and the url as the value with a TTL of 300 seconds (5 min).  Default
	will be	172800 Seconds (48 hrs) in production. */
	if err := c.Set(fullURL, code, time.Duration(300*time.Second)).Err(); err != nil {
		return err
	}

	return nil
}

// generates a unqiue code and checks if valid
func generate(c *redis.Client, n int) (string, error) {
	code := GenCode(n)
	exists := c.Exists(code).Val()

	if exists == 0 {
		return code, nil
	}

	log.Info("Redis Discover:", "Key Collision", "Generating new code")
	return generate(c, n+1)
}
