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
// This does not interfere with the normal byte stream passing over the unix file socketv

package oob_test

import (
	"context"
	"encoding/binary"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/edwarnicke/exechelper"

	"github.com/edwarnicke/oob"
)

func TestUnixsocket_RecvFile(t *testing.T) {
	dirname, err := ioutil.TempDir(os.TempDir(), "oob_test")
	require.NoError(t, err)
	socketfilename := filepath.Join(dirname, "socket")
	listener, err := net.Listen("unix", socketfilename)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())

	err = exechelper.Run("go build .",
		exechelper.WithDir("./testdata/sendfile"),
		exechelper.WithStdout(os.Stdout),
		exechelper.WithStderr(os.Stderr),
	)
	require.NoError(t, err)
	errCh := exechelper.Start("./sendfile",
		exechelper.WithArgs(socketfilename),
		exechelper.WithDir("./testdata/sendfile"),
		exechelper.WithStdout(os.Stdout),
		exechelper.WithStderr(os.Stderr),
		exechelper.WithContext(ctx),
	)
	conn, err := listener.Accept()
	require.NoError(t, err)
	defer func() { assert.NoError(t, conn.Close()) }()
	o := oob.New(conn.(*net.UnixConn))
	for i := 0; i < 3; i++ {
		fd, err := o.RecvFD()
		// Only 2 file descriptors are sent, so on the third, we expect EINVAL
		if i == 2 && err != nil && err.(syscall.Errno) == syscall.EINVAL {
			continue
		}
		require.NoError(t, err)
		file := oob.ToFile(fd)
		fi, err := os.Stat(file.Name())
		require.NoError(t, err)
		buf := make([]byte, fi.Size())
		n, err := file.ReadAt(buf, 0)
		require.NoError(t, err)
		assert.EqualValues(t, n, fi.Size())
		x, n := binary.Varint(buf)
		assert.EqualValues(t, n, fi.Size())

		inode, err := oob.ToInode(fd)
		require.NoError(t, err)
		assert.EqualValues(t, inode, x)
	}
	cancel()
	for err := range errCh {
		assert.IsType(t, &exec.ExitError{}, err) // Because we canceled we will get an exec.ExitError{}
		assert.Empty(t, errCh)
		assert.Zero(t, err.(*exec.ExitError).ExitCode())
	}
}

func TestUnixsocket_SendSocket(t *testing.T) {
	// Open a socket over which we pass FDs
	dirname, err := ioutil.TempDir(os.TempDir(), "oob_test")
	require.NoError(t, err)
	socketfilename := filepath.Join(dirname, "socket")
	listener, err := net.Listen("unix", socketfilename)
	require.NoError(t, err)

	// Build and run test binaries
	err = exechelper.Run("go build .",
		exechelper.WithDir("./testdata/recvsocket"),
		exechelper.WithStdout(os.Stdout),
		exechelper.WithStderr(os.Stderr),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	errCh := exechelper.Start("./recvsocket",
		exechelper.WithArgs(socketfilename),
		exechelper.WithDir("./testdata/recvsocket"),
		exechelper.WithStdout(os.Stdout),
		exechelper.WithStderr(os.Stderr),
		exechelper.WithContext(ctx),
	)

	// Accept connection from test binary
	conn, err := listener.Accept()
	require.NoError(t, err)
	defer func() { assert.NoError(t, conn.Close()) }()
	o := oob.New(conn.(*net.UnixConn))

	// Create a test server
	socketFilenameUnderTest := filepath.Join(dirname, "socketUnderTest")
	listenerUnderTest, err := net.Listen("unix", socketFilenameUnderTest)
	require.NoError(t, err)

	// Capture the incoming connection to the test server *from ourselves*
	incomingCh := make(chan net.Conn)
	go func(listenerUnderTest net.Listener) {
		incoming, incomingErr := listenerUnderTest.Accept()
		require.NoError(t, incomingErr)
		incomingCh <- incoming
	}(listenerUnderTest)

	// Connect to ourselves
	connUnderTest, err := (&net.Dialer{}).DialContext(ctx, "unix", socketFilenameUnderTest)
	require.NoError(t, err)
	fd, err := oob.ToFd(connUnderTest)
	require.NoError(t, err)

	// Set that active socket to sendsocket
	err = o.SendFD(fd)
	require.NoError(t, err)

	// Get the incoming side of that connection
	incoming := <-incomingCh
	defer func() { assert.NoError(t, incoming.Close()) }()

	// Get the inode of the socket
	inode, err := oob.ToInode(fd)
	require.NoError(t, err)

	// Read the size of the inode from the socket
	buf := make([]byte, binary.Size(inode))
	_, err = incoming.Read(buf)
	require.NoError(t, err)

	// Check to see we got the expected inode from the other side
	x, _ := binary.Varint(buf)
	assert.EqualValues(t, inode, x)
	cancel()
	for err := range errCh {
		assert.IsType(t, &exec.ExitError{}, err) // Because we canceled we will get an exec.ExitError{}
		assert.Empty(t, errCh)
		assert.Zero(t, err.(*exec.ExitError).ExitCode())
	}
}
