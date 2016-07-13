/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"os"
	"runtime"

	l "github.com/ernestio/builder-library"
)

var s l.Scheduler

func main() {
	s.Setup(os.Getenv("NATS_URI"))

	s.ProcessRequest("networks.create", "network.create")
	s.ProcessRequest("networks.delete", "network.delete")

	s.ProcessSuccessResponse("network.create.done", "network.create", "networks.create.done")
	s.ProcessSuccessResponse("network.delete.done", "network.delete", "networks.delete.done")

	s.ProcessFailedResponse("network.create.error", "networks.create.error")
	s.ProcessFailedResponse("network.delete.error", "networks.delete.error")

	runtime.Goexit()
}
