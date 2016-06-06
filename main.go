/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import "runtime"

func main() {
	n := natsClient()
	r := redisClient()

	// Process requests
	processRequest(n, r, "networks.create", "network.create")
	processRequest(n, r, "networks.delete", "network.delete")

	// Process resulting success
	processResponse(n, r, "network.create.done", "networks.create.", "network.create", "completed")
	processResponse(n, r, "network.delete.done", "networks.delete.", "network.delete", "completed")

	// Process resulting errors
	processResponse(n, r, "network.create.error", "networks.create.", "network.create", "errored")
	processResponse(n, r, "network.delete.error", "networks.delete.", "network.delete", "errored")

	runtime.Goexit()
}
