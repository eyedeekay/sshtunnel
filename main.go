// Copyright 2016, The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// sshtunnel is daemon for setting up forward and reverse SSH tunnels.
//
// The daemon is started by executing sshtunnel with the path to a JSON
// configuration file. The configuration takes the following form:
//
//	{
//		"KeyFiles": ["/path/to/key.priv"],
//		"KnownHostFiles": ["/path/to/known_hosts"],
//		"Tunnels": [{
//			// Forward tunnel (locally binded socket proxies to remote target).
//			"Tunnel": "bind_address:port -> dial_address:port",
//			"Server": "user@host:port",
//		}, {
//			// Reverse tunnel (remotely binded socket proxies to local target).
//			"Tunnel": "dial_address:port <- bind_address:port",
//			"Server": "user@host:port",
//		}],
//	}
//
// See the TunnelConfig struct for more details.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	. "github.com/dsnet/sshtunnel/tunnel"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "\t%s CONFIG_PATH\n", os.Args[0])
		os.Exit(1)
	}
	tunns, logger, closer := LoadConfig(os.Args[1])
	defer closer()

	// Setup signal handler to initiate shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		logger.Printf("received %v - initiating shutdown", <-sigc)
		cancel()
	}()

	// Start a bridge for each tunnel.
	var wg sync.WaitGroup
	logger.Printf("%s starting", path.Base(os.Args[0]))
	defer logger.Printf("%s shutdown", path.Base(os.Args[0]))
	for _, t := range tunns {
		wg.Add(1)
		go t.BindTunnel(ctx, &wg)
	}
	wg.Wait()
}
