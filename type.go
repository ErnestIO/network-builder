/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"gopkg.in/redis.v3"
)

// NetworksCreate : Represents a networks.create message
type NetworksCreate struct {
	Service              string    `json:"service"`
	Status               string    `json:"status"`
	ErrorCode            string    `json:"error_code"`
	ErrorMessage         string    `json:"error_message"`
	Networks             []network `json:"networks"`
	SequentialProcessing bool      `json:"sequential_processing"`
}

func (e *NetworksCreate) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

func (e *NetworksCreate) cacheKey() string {
	return composeCacheKey(e.Service)
}

func composeCacheKey(service string) string {
	var key bytes.Buffer
	key.WriteString("GPBNetworks_")
	key.WriteString(service)

	return key.String()
}

type network struct {
	Type               string   `json:"type"`
	Name               string   `json:"name"`
	Range              string   `json:"range"`
	Subnet             string   `json:"subnet,omitempty"`
	Netmask            string   `json:"netmask,omitempty"`
	StartAddress       string   `json:"start_address,omitempty"`
	EndAddress         string   `json:"end_address,omitempty"`
	Gateway            string   `json:"gateway,omitempty"`
	DNS                []string `json:"dns"`
	Router             string   `json:"router"`
	RouterType         string   `json:"router_type"`
	RouterName         string   `json:"router_name,omitempty"`
	ClientName         string   `json:"client_name"`
	DatacenterType     string   `json:"datacenter_type,omitempty"`
	DatacenterName     string   `json:"datacenter_name,omitempty"`
	DatacenterUsername string   `json:"datacenter_username,omitempty"`
	DatacenterPassword string   `json:"datacenter_password,omitempty"`
	VCloudURL          string   `json:"vcloud_url"`
	Status             string   `json:"status"`
	ErrorCode          string   `json:"error_code"`
	ErrorMessage       string   `json:"error_message"`
}

func (n *network) fail() {
	n.Status = "errored"
}

func (n *network) complete() {
	n.Status = "completed"
}

func (n *network) processing() {
	n.Status = "processed"
}

func (n *network) errored() bool {
	return n.Status == "errored"
}

func (n *network) completed() bool {
	println(n.Status)
	return n.Status == "completed"
}

func (n *network) isProcessed() bool {
	return n.Status == "processed"
}

func (n *network) isErrored() bool {
	return n.Status == "errored"
}

func (n *network) toBeProcessed() bool {
	return n.Status != "processed" && n.Status != "completed" && n.Status != "errored"
}

func (n *network) Valid() (bool, string) {
	if n.Name == "" {
		return false, "Network name can not be empty"
	}
	if n.DatacenterName == "" {
		return false, "Specifying a datacenter is necessary when creating a network"
	}

	return true, ""
}

type networks struct {
	Collection []network
}

type networkEvent struct {
	Type                string   `json:"type"`
	Service             string   `json:"service"`
	NetworkType         string   `json:"network_type"`
	NetworkName         string   `json:"network_name"`
	NetworkNetmask      string   `json:"network_netmask"`
	NetworkStartAddress string   `json:"network_start_address"`
	NetworkEndAddress   string   `json:"network_end_address"`
	NetworkGateway      string   `json:"network_gateway"`
	DNS                 []string `json:"network_dns"`
	RouterName          string   `json:"router_name"`
	RouterType          string   `json:"router_type,omitempty"`
	RouterIP            string   `json:"router_ip"`
	ClientName          string   `json:"client_name,omitempty"`
	DatacenterType      string   `json:"datacenter_type,omitempty"`
	DatacenterName      string   `json:"datacenter_name,omitempty"`
	DatacenterUsername  string   `json:"datacenter_username,omitempty"`
	DatacenterPassword  string   `json:"datacenter_password,omitempty"`
	DatacenterRegion    string   `json:"datacenter_region,omitempty"`
	VCloudURL           string   `json:"vcloud_url"`
}

func (e *networkEvent) load(nw network, t string, s string) {
	e.Service = s
	e.Type = t
	e.RouterName = nw.RouterName
	e.RouterType = nw.RouterType
	e.NetworkType = nw.RouterType
	e.NetworkName = nw.Name
	e.ClientName = nw.ClientName
	e.DatacenterName = nw.DatacenterName
	e.DatacenterUsername = nw.DatacenterUsername
	e.DatacenterPassword = nw.DatacenterPassword
	e.DatacenterType = nw.DatacenterType
	e.VCloudURL = nw.VCloudURL
	e.DNS = nw.DNS

	octets := getIPOctets(nw.Range)
	e.NetworkNetmask = nw.ParseNetmask()
	e.NetworkStartAddress = octets + ".5"
	e.NetworkEndAddress = octets + ".250"
	e.NetworkGateway = octets + ".1"
}

func (e *networkEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

// Error ...
type Error struct {
	Code    json.Number `json:"code,Number"`
	Message string      `json:"message"`
}

type networkCreatedEvent struct {
	Type        string `json:"type"`
	Service     string `json:"service"`
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	Error       Error  `json:"error"`
}

func (n *network) ParseNetmask() string {
	// Convert netmask hex to string, generated from network range CIDR
	_, nw, _ := net.ParseCIDR(n.Range)
	hx, _ := hex.DecodeString(nw.Mask.String())
	netmask := fmt.Sprintf("%v.%v.%v.%v", hx[0], hx[1], hx[2], hx[3])
	return netmask
}

func getIPOctets(rng string) string {
	// Splits the network range and returns the first three octets
	ip, _, err := net.ParseCIDR(rng)
	if err != nil {
		log.Println(err)
	}
	octets := strings.Split(ip.String(), ".")
	octets = append(octets[:3], octets[3+1:]...)
	octetString := strings.Join(octets, ".")
	return octetString
}

func (e *networkCreatedEvent) cacheKey() string {
	return composeCacheKey(e.Service)
}

func persistEvent(redisClient *redis.Client, event *NetworksCreate) {
	if event.Service == "" {
		panic("Service is null!")
	}
	if err := redisClient.Set(event.cacheKey(), event.toJSON(), 0).Err(); err != nil {
		log.Println(err)
	}
}
