/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

var setup = false
var n *nats.Conn
var r *redis.Client

type DummyEvent struct {
	Type string `json:"type"`
	Name string `json:"network_name"`
}

func wait(ch chan bool) error {
	return waitTime(ch, 500*time.Millisecond)
}

func waitTime(ch chan bool, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
	}
	return errors.New("timeout")
}

func testSetup() {
	if setup == false {
		os.Setenv("NATS_URI", "nats://localhost:4222")

		n = natsClient()
		n.Subscribe("config.get.redis", func(msg *nats.Msg) {
			n.Publish(msg.Reply, []byte(`{"DB":0,"addr":"localhost:6379","password":""}`))
		})
		r = redisClient()
		setup = true
	}
}

func TestProvisionAllNetworksBasic(t *testing.T) {
	testSetup()
	processRequest(n, r, "networks.create", "network.provision")

	ch := make(chan bool)

	n.Subscribe("network.provision", func(ev *nats.Msg) {
		event := networkEvent{}
		json.Unmarshal(ev.Data, &event)
		if event.Type == "network.provision" &&
			event.NetworkType == "vcloud" &&
			event.DNS[0] == "8.8.8.8" &&
			event.NetworkName == "test" {
			eventKey := "GPBNetworks_" + event.Service
			message, _ := r.Get(eventKey).Result()
			stored := &NetworksCreate{}
			json.Unmarshal([]byte(message), stored)
			if stored.Service != event.Service {
				t.Fatal("Event is not persisted correctly")
			}
			ch <- true
		} else {
			t.Fatal("Message received from nats does not match")
		}
	})

	message := `{"service":"service", "networks":[{"name":"test","range":"10.64.0.0/24","router":"test", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test", "dns": ["8.8.8.8"]}]}`
	n.Publish("networks.create", []byte(message))
	time.Sleep(100 * time.Millisecond)

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}

}

func TestProvisionAllNetworksWithInvalidMessage(t *testing.T) {
	testSetup()

	processRequest(n, r, "networks.create", "network.provision")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("networks.create.error", func(msg *nats.Msg) {
		ch2 <- true
	})

	message := `{"service":"service", "networks":[{"name":"test","network":"test"}]}`
	n.Publish("networks.create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a network.provision message when I shouldn't")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Should produce a provision-all-networks-error message on nats")
	}
}

func TestProvisionAllNetworksWithInvalidCIDR(t *testing.T) {
	testSetup()
	processRequest(n, r, "networks.create", "network.provision")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("networks.create.error", func(msg *nats.Msg) {
		ch2 <- true
	})

	message := `{"service":"service", "networks":[{"name":"test","network":"test", "range":"10.64.0.0/99"}]}`
	n.Publish("networks.create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a network.provision message when I shouldn't")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Should produce a provision-all-networks-error message on nats")
	}
}

func TestProvisionAllNetworksSendingTwoNetworks(t *testing.T) {
	testSetup()

	processRequest(n, r, "networks.create", "network.provision")

	ch := make(chan bool)

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		event := networkEvent{}
		json.Unmarshal(msg.Data, &event)
		if event.Type == "network.provision" && event.NetworkName == "test2" {
			ch <- true
		}
	})

	message := `{"service":"service", "sequential_processing": true, "networks":[{"name":"test","range":"10.64.0.0/24","router":"test", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test", "dns":["8.8.8.8"]},{"name":"test2","range":"10.64.0.0/24","router":"test2", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test", "dns":["8.8.8.8"]}]}`
	n.Publish("networks.create", []byte(message))
	time.Sleep(100 * time.Millisecond)

	if e := wait(ch); e == nil {
		t.Fatal("Second Message received from NATS!")
	}
}

func TestProvisionAllnetworksWithDifferentMessageType(t *testing.T) {
	testSetup()
	processRequest(n, r, "networks.create", "network.provision")

	ch := make(chan bool)

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		ch <- true
	})

	message := `{"service":"service", "routers":[{"name":"test"}]}`
	n.Publish("networks.create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a network.provision message when I shouldn't")
	}
}

func TestHandleNetworkCompletedEvent(t *testing.T) {
	testSetup()

	processResponse(n, r, "network.create.done", "networks.create.", "network.provision", "completed")

	// Add record to Redis
	message := `{"service":"service", "networks":[{"name":"test","range":"10.64.0.0/24","router":"test", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"},{"name":"test2","range":"10.64.0.0/24","router":"test2", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"}]}`
	r.Set("GPBNetworks_service", message, 0)

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("networks.create.error", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		ev := DummyEvent{}
		json.Unmarshal(msg.Data, &ev)
		if ev.Type == "network.provision" {
			ch2 <- true
		}
	})

	ev := `{"type": "network.create.done", "service": "service", "network_id": "1", "network_name": "test", "router_id": "test", "network_type":"vcloud", "network_range":"10.64.0.0/24"}`
	n.Publish("network.create.done", []byte(ev))

	// Should receive a provision event
	if e := wait(ch); e == nil {
		t.Fatal("Produced an error when i shouldn't have")
	}

	if e := wait(ch2); e != nil {
		t.Fatal("Didn't produce a network.provision event when i should have")
	}
}

func TestHandleNextInSequenceEvent(t *testing.T) {
	testSetup()

	processResponse(n, r, "network.create.done", "networks.create.", "network.provision", "completed")

	// Add record to Redis
	message := `{"service":"service", "networks":[{"name":"test","range":"10.64.0.0/24","router":"test", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"},{"name":"test2","range":"10.64.0.0/24","router":"test2", "router_name": "test",	"router_type": "vcloud", "router_ip": "8.8.8.24", "datacenter": "test", "datacenter_name": "test",	"datacenter_type": "vcloud", "datacenter_region": "LON-001", "datacenter_username": "test@test", "datacenter_password": "test", "client_id": "test", "client_name": "test"}]}`
	r.Set("GPBNetworks_service", message, 0)

	ch := make(chan bool)
	ch2 := make(chan []byte)

	n.Subscribe("networks.create.error", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("network.provision", func(msg *nats.Msg) {
		ev := DummyEvent{}
		json.Unmarshal(msg.Data, &ev)
		if ev.Type == "network.provision" {
			ch2 <- msg.Data
		}
	})

	ev := `{"type": "network.create.done", "service": "service", "network_id": "1", "network_name": "test", "router_id": "test", "network_type":"vcloud", "network_range":"10.64.0.0/24"}`
	n.Publish("network.create.done", []byte(ev))

	if e := wait(ch); e == nil {
		t.Fatal("Produced an error when i shouldn't have")
	}

	// Should receive next provision event in sequence
	nev := <-ch2
	nextEvent := networkEvent{}
	json.Unmarshal(nev, &nextEvent)

	if nextEvent.Service == "service" && nextEvent.NetworkName == "test2" && nextEvent.Type == "network.provision" {
		log.Println("Correct event received")
	} else {
		t.Fatal("Did not produce the correct next event")
	}
}

func TestHandleFinalEvent(t *testing.T) {
	testSetup()

	processResponse(n, r, "network.create.done", "networks.create.", "network.provision", "completed")

	// Add record to Redis
	message := `{"service":"service","networks":[{"name":"test","range":"10.64.0.0/24","netmask":"255.255.255.0","start_address":"10.64.0.5","end_address":"10.64.0.250","gateway":"10.64.0.1","router":"test","router_name":"test","router_type":"vcloud","router_ip":"8.8.8.24","client_id":"test","client_name":"test","datacenter":"test","datacenter_type":"vcloud","datacenter_name":"test","datacenter_username":"test@test","datacenter_password":"test","datacenter_region":"LON-001"}]}`
	r.Set("GPBNetworks_service", message, 0)

	ch := make(chan bool)

	n.Subscribe("networks.create.done", func(msg *nats.Msg) {
		ch <- true
	})

	ev := `{"type": "network.create.done", "service": "service", "network_id": "1", "network_name": "test", "router_id": "test", "network_type":"vcloud", "network_range":"10.64.0.0/24"}`
	n.Publish("network.create.done", []byte(ev))

	// Should receive next provision event in sequence
	if e := wait(ch); e != nil {
		t.Fatal("Did not produce a completed event")
	}
}
