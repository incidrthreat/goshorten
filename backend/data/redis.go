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
func (r *Redis) Save(url string, ttl int64) (string, error) {
	// discover returns a uniqe code doesnt exist
	code, err := generate(r.Client, r.CharFloor)
	if err != nil {
		log.Error("Redis Save Error", "Generate code", hclog.Fmt("%v", err))
		return "", err
	}
	err = set(r.Client, url, code, ttl)
	if err != nil {
		log.Error("Redis Set Error", "Details", hclog.Fmt("%v", err))
		return "", err
	}
	log.Info("Redis Save", "Code stored", hclog.Fmt("Code: %s | URL: %s | TTL: %d Seconds", code, url, ttl))
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
func set(c *redis.Client, code string, fullURL string, ttl int64) error {
	if err := c.Set(fullURL, code, time.Duration(ttl)*time.Second).Err(); err != nil {
		return err
	}
	return nil
}

// generates a unqiue code and checks if valid
func generate(c *redis.Client, n int) (string, error) {
	code := GenCode(n)
	exists := c.Exists(code).Val()

	// genAttempts and the below for loop provide 3 tries before exiting with too many collisions detected.
	// Thanks @maikthulhu for catching this and suggestion that 0 + 0 indeed equals 0. <3
	genAttempts := 3
	for exists != 0 && n <= 6 && genAttempts > 0 {
		log.Warn("Redis Warning", "Key Collision", hclog.Fmt("Collision on code: %s, generating new code.", code))
		code = GenCode(n + 1)
		exists = c.Exists(code).Val()
		genAttempts--
	}

	if genAttempts == 0 {
		return "", errors.New("3 code collisions detected, try again later")
	}

	return code, nil
}
