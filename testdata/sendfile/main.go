// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package oob - Simple out of band file descriptor passing over Unix File Sockets
// Linux allows the passing of file descriptors out of band over unix file sockets
// This does not interfere with the normal byte stream passing over the unix file socket

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/edwarnicke/oob"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", os.Args[1])
	exitOnErr(err)
	defer conn.Close()
	o := oob.New(conn.(*net.UnixConn))
	for i := 0; i < 2; i++ {
		file, err := ioutil.TempFile(os.TempDir(), "oob-file")
		exitOnErr(err)
		defer file.Close()
		fi, err := file.Stat()
		exitOnErr(err)
		inode := fi.Sys().(*syscall.Stat_t).Ino
		buf := make([]byte, binary.Size(inode))
		n := binary.PutVarint(buf, int64(inode))
		b := buf[:n]
		file.Write(b)

		exitOnErr(o.SendFD(file.Fd()))
	}
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}
