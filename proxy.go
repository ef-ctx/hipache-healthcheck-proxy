// Copyright 2016 EF CTX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const version = "0.1.2"

var printVersion bool

func init() {
	flag.BoolVar(&printVersion, "v", false, "print version and exit")
	flag.Parse()
}

type config struct {
	BindAddress    string `envconfig:"BIND_ADDRESS" default:":9000"`
	HipacheAddress string `envconfig:"HIPACHE_ADDRESS" required:"true"`
}

func main() {
	if printVersion {
		fmt.Printf("hipache-healthcheck-proxy %s\n", version)
		return
	}
	var c config
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = url.Parse(c.HipacheAddress); err != nil {
		log.Fatalf("failed to parse hipache address: %s", err)
	}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: time.Second}).DialContext,
		},
		Timeout: 2 * time.Second,
	}

	log.Printf("starting on %s...", c.BindAddress)
	err = http.ListenAndServe(c.BindAddress, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest("GET", c.HipacheAddress, nil)
		if err != nil {
			http.Error(w, "failed to process request", http.StatusInternalServerError)
			return
		}
		req.Host = "__ping__"
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		for h := range resp.Header {
			w.Header().Set(h, resp.Header.Get(h))
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	if err != nil {
		log.Fatal(err)
	}
}
