/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"os"

	ecc "github.com/ernestio/ernest-config-client"
	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

var c *ecc.Config

// natsClient : creates a new nats client
func natsClient() *nats.Conn {
	c = ecc.NewConfig(os.Getenv("NATS_URI"))
	return c.Nats()
}

// redisClient : creates a redis client
func redisClient() *redis.Client {
	return c.Redis()
}
