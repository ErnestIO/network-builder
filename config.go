/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

// RedisConfig : struct representation of service configuration
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int64  `json:"db"`
}

// natsClient : creates a new nats client
func natsClient() *nats.Conn {
	natsURI := os.Getenv("NATS_URI")
	if natsURI == "" {
		natsURI = nats.DefaultURL
	}

	n, err := nats.Connect(natsURI)
	if err != nil {
		log.Println("Could not connect to NATS server")
		panic(err)
	}

	return n
}

// redisClient : creates a redis client
func redisClient() *redis.Client {
	cfg := RedisConfig{}

	if os.Getenv("REDIS_ADDR") != "" {
		return redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})
	}

	// Get config from conf-store
	nc := natsClient()

	resp, err := nc.Request("config.get.redis", nil, time.Second)
	if err != nil {
		log.Println("could not load config")
		log.Panic(err)
	}

	err = json.Unmarshal(resp.Data, &cfg)
	if err != nil {
		log.Panic(err)
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
