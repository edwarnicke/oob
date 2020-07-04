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
	"net"
	"os"
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

	fd, err := o.RecvFD()
	exitOnErr(err)
	socketConn, err := oob.ToConn(fd)
	exitOnErr(err)
	defer socketConn.Close()

	socketInode, err := oob.ToInode(fd)
	exitOnErr(err)
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutVarint(buf, int64(socketInode))
	b := buf[:n]
	n, err = socketConn.Write(b)
	exitOnErr(err)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
