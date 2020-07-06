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

package oob_test

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/edwarnicke/oob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInodeToInode(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fi, err := file.Stat()
	assert.NoError(t, err)
	inode := fi.Sys().(*syscall.Stat_t).Ino
	inode2, err := oob.ToInode(inode)
	assert.NoError(t, err)
	assert.Equal(t, inode, inode2)
}

func TestFileToInode(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fi, err := file.Stat()
	assert.NoError(t, err)
	inode := fi.Sys().(*syscall.Stat_t).Ino
	inode2, err := oob.ToInode(file)
	assert.NoError(t, err)
	assert.Equal(t, inode, inode2)
}

func TestFdToInode(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fi, err := file.Stat()
	assert.NoError(t, err)
	inode := fi.Sys().(*syscall.Stat_t).Ino
	inode2, err := oob.ToInode(file.Fd())
	assert.NoError(t, err)
	assert.Equal(t, inode, inode2)
}

func TestFdToFd(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fd := file.Fd()
	fd2, err := oob.ToFd(fd)
	assert.NoError(t, err)
	assert.Equal(t, fd, fd2)
}

func TestFileToFd(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fd := file.Fd()
	fd2, err := oob.ToFd(file)
	assert.NoError(t, err)
	assert.Equal(t, fd, fd2)
}

func TestInodeToFd(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	fi, err := file.Stat()
	assert.NoError(t, err)
	inode := fi.Sys().(*syscall.Stat_t).Ino
	fd, err := oob.ToFd(inode)
	require.NoError(t, err)
	assert.Equal(t, file.Fd(), fd)
}

func TestFileToFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "oob-fileTofile")
	assert.NoError(t, err)
	file2, err := oob.ToFile(file)
	assert.NoError(t, err)
	assert.Equal(t, file, file2)
}

func CreatTestConn(t *testing.T) net.Conn {
	// Create a test server
	dirname, err := ioutil.TempDir(os.TempDir(), "oob_test")
	require.NoError(t, err)
	socketfilename := filepath.Join(dirname, "socketUnderTest")
	listener, err := net.Listen("unix", socketfilename)
	require.NoError(t, err)

	// Capture the incoming connection to the test server *from ourselves*
	incomingCh := make(chan net.Conn, 1)
	go func(listener net.Listener) {
		incoming, incomingErr := listener.Accept()
		require.NoError(t, incomingErr)
		incomingCh <- incoming
	}(listener)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socketfilename)
	require.NoError(t, err)
	return conn
}

func TestConnToConn(t *testing.T) {
	conn := CreatTestConn(t)
	conn2, err := oob.ToConn(conn)
	require.NoError(t, err)
	assert.Equal(t, conn, conn2)
}

func TestFileToConn(t *testing.T) {
	conn := CreatTestConn(t)
	file, err := oob.ToFile(conn)
	require.NoError(t, err)
	conn2, err := oob.ToConn(file)
	require.NoError(t, err)
	inode, err := oob.ToInode(conn)
	require.NoError(t, err)
	require.NoError(t, err)
	inode2, err := oob.ToInode(conn2)
	require.NoError(t, err)
	assert.Equal(t, inode2, inode)
}

func TestFdToConn(t *testing.T) {
	conn := CreatTestConn(t)
	fd, err := oob.ToFd(conn)
	require.NoError(t, err)
	conn2, err := oob.ToConn(fd)
	require.NoError(t, err)
	inode, err := oob.ToInode(conn)
	require.NoError(t, err)
	inode2, err := oob.ToInode(conn2)
	require.NoError(t, err)
	assert.Equal(t, inode2, inode)
}

func TestInodeToConn(t *testing.T) {
	conn := CreatTestConn(t)
	inode, err := oob.ToInode(conn)
	require.NoError(t, err)
	conn2, err := oob.ToConn(inode)
	require.NoError(t, err)
	inode2, err := oob.ToInode(conn2)
	require.NoError(t, err)
	assert.Equal(t, inode2, inode)
}
