/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

func updateNetwork(n *nats.Conn, network network, s string, t string) {
	e := networkEvent{}
	e.load(network, t, s)
	n.Publish(t, []byte(e.toJSON()))
}

func processNext(n *nats.Conn, r *redis.Client, subject string, procSubject string, body []byte, status string) (*NetworksCreate, bool) {
	event := &networkCreatedEvent{}
	json.Unmarshal(body, event)

	message, err := r.Get(event.cacheKey()).Result()
	if err != nil {
		log.Println(err)
	}
	stored := &NetworksCreate{}
	err = json.Unmarshal([]byte(message), stored)
	if err != nil {
		log.Println(err)
	}
	completed := true
	scheduled := false
	for i := range stored.Networks {
		if stored.Networks[i].Name == event.NetworkName {
			stored.Networks[i].Status = status
			if stored.Networks[i].errored() == true {
				stored.Networks[i].ErrorCode = string(event.Error.Code)
				stored.Networks[i].ErrorMessage = event.Error.Message
			}
		}
		if stored.Networks[i].completed() == false && stored.Networks[i].errored() == false {
			completed = false
		}
		if stored.Networks[i].toBeProcessed() && scheduled == false {
			scheduled = true
			completed = false
			stored.Networks[i].processing()
			updateNetwork(n, stored.Networks[i], event.Service, procSubject)
		}
	}
	persistEvent(r, stored)

	return stored, completed
}

func processResponse(n *nats.Conn, r *redis.Client, s string, res string, p string, t string) {
	n.Subscribe(s, func(m *nats.Msg) {
		stored, completed := processNext(n, r, s, p, m.Data, t)
		if completed {
			complete(n, stored, res)
		}
	})
}

func complete(n *nats.Conn, stored *NetworksCreate, subject string) {
	if isErrored(stored) == true {
		stored.Status = "error"
		stored.ErrorCode = "0002"
		stored.ErrorMessage = "Some networks could not been successfully processed"
		n.Publish(subject+"error", []byte(stored.toJSON()))
	} else {
		stored.Status = "completed"
		stored.ErrorCode = ""
		stored.ErrorMessage = ""
		n.Publish(subject+"done", []byte(stored.toJSON()))
	}
}

func isErrored(stored *NetworksCreate) bool {
	for _, v := range stored.Networks {
		if v.isErrored() {
			return true
		}
	}
	return false
}

func processRequest(n *nats.Conn, r *redis.Client, subject string, resSubject string) {
	n.Subscribe(subject, func(m *nats.Msg) {
		event := NetworksCreate{}
		json.Unmarshal(m.Data, &event)

		persistEvent(r, &event)

		if len(event.Networks) == 0 || event.Status == "completed" {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}

		for _, network := range event.Networks {
			if ok, msg := network.Valid(); ok == false {
				event.Status = "error"
				event.ErrorCode = "0001"
				event.ErrorMessage = msg
				n.Publish(subject+".error", []byte(event.toJSON()))
				return
			}
		}

		sw := false
		for i, network := range event.Networks {
			if event.Networks[i].completed() == false {
				sw = true
				event.Networks[i].processing()
				updateNetwork(n, network, event.Service, resSubject)
				if true == event.SequentialProcessing {
					break
				}
			}
		}

		if sw == false {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}

		persistEvent(r, &event)
	})
}
