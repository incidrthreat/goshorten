package data

import (
	"encoding/json"
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
func (r *Redis) Save(url string, ttl int64, stats string) (string, error) {
	// generate returns a unique code that doesnt exist
	code, err := generate(r.Client, r.CharFloor)
	if err != nil {
		log.Error("Redis Save Error", "Generate code", hclog.Fmt("%v", err))
		return "", err
	}

	// Sets URL, Code, and TTL for single input
	_, err = r.Client.Set(code, url, time.Duration(ttl)*time.Second).Result()
	if err == redis.Nil {
		return "", errors.New("Cannot Save URL")
	} else if err != nil {
		log.Error("Redis URL Save", "Error", err)
		return "", err
	}

	statMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(stats), &statMap)

	// Saves the stats as a hash with the key as Code:<code>
	_, err = r.Client.HMSet("Code:"+code, statMap).Result()
	if err == redis.Nil {
		return "", errors.New("Cannot Save Stats")
	} else if err != nil {
		log.Error("Redis Stats Save", "Error", err)
		return "", err
	}

	log.Info("Redis Save", "Code stored", hclog.Fmt("Code: %s | URL: %s | TTL: %d Seconds", code, url, ttl))
	return code, nil
}

// Load grabs data from redis
func (r Redis) Load(code string) (string, error) {
	fullURL, err := r.Client.Get(code).Result()

	if err == redis.Nil {
		return "", errors.New("Code not found")
	} else if err != nil {
		log.Error("Redis Load", "Error", err)
		return "", err
	}

	// incriments "clicks" value
	_, err = r.Client.HIncrBy("Code:"+code, "clicks", 1).Result()
	if err == redis.Nil {
		log.Error("Error", "Code does not exist", err)
	} else if err != nil {
		log.Error("Error", "Increment Failed", err)
	}

	// updates "last_accessed" value
	_, err = r.Client.HSet("Code:"+code, "last_accessed", time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST")).Result()
	if err == redis.Nil {
		log.Error("Error", "Code does not exist", err)
	} else if err != nil {
		log.Error("Error", "Last Accessed value failed to set", err)
	}

	log.Info("Redis Load", "URL retrieved", hclog.Fmt("%s", fullURL))
	return fullURL, nil
}

// Stats retrieves current stats on a code.
func (r Redis) Stats(code string) (string, error) {
	stats, err := r.Client.HGetAll("Code:" + code).Result()
	url, _ := r.Client.Get(code).Result()

	if err == redis.Nil {
		log.Error("Error", "Code does not exist", err)
		return "", err
	} else if err != nil {
		log.Error("Error", "Retrieval Failed", err)
		return "", err
	}

	var statsMap = make(map[string]string)
	for k, v := range stats {
		statsMap[k] = v
	}
	statsMap["code"] = code
	statsMap["url"] = url

	statsJSON, err := json.Marshal(statsMap)
	if err != nil {
		log.Error("Error", "Could not marshal stats", err)
		return "", err
	}
	return string(statsJSON), nil
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
