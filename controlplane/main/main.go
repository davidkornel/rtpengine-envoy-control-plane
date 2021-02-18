// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
package main

import (
	"flag"
	"sync"

	cp "rtpengine-envoy-control-plane/controlplane"
)

var (
	l cp.Logger

	port uint

	nodeID string
)

func init() {
	l = cp.Logger{}

	flag.BoolVar(&l.Debug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")
}

func main() {
	flag.Parse()

	//new waitgroup
	wg := new(sync.WaitGroup)
	wg.Add(2)

	// Run the xDS server
	go func() {
		cp.RunServer(port, &l)
		wg.Done()
	}()

	//Run the UDP server
	go func() {
		cp.Server(":1234", &l)
		wg.Done()
	}()

	wg.Wait()
}
