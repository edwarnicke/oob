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
package oob

import (
	"fmt"
	"net"
	"os"

	"syscall"

	"github.com/pkg/errors"
)

// UnixConnOob - net.UnixConn + SendFD and RecvFD methods for sending and receiving file descriptors
type UnixConnOob struct {
	*net.UnixConn
}

// New - wrap a *net.UnixConn providing it additional methods to SendFD and RecvFD
func New(s *net.UnixConn) *UnixConnOob {
	return &UnixConnOob{s}
}

// SendFD - send the file descriptor fd to the process on the other end of the *net.UnixConn
func (s *UnixConnOob) SendFD(fd uintptr) error {
	socketFile, err := s.UnixConn.File()
	if err != nil {
		return err
	}
	rights := syscall.UnixRights(int(fd))
	err = syscall.Sendmsg(int(socketFile.Fd()), nil, rights, nil, 0)
	if err != nil {
		return err
	}
	return nil
}

// RecvFD - recv a file descriptor over a *net.UnixConn
//       But you usually can't *link* it to another file location due to cross device errors
// Note: If you  call s.RecvFD() when no fd is available, it will return error syscall.Errno == syscall.EINVAL
func (s *UnixConnOob) RecvFD() (fd uintptr, err error) {
	socketFile, err := s.UnixConn.File()
	if err != nil {
		return 0, err
	}
	buf := make([]byte, syscall.CmsgSpace(4))
	_, _, _, _, err = syscall.Recvmsg(int(socketFile.Fd()), nil, buf, 0)
	if err != nil {
		return 0, err
	}
	var msgs []syscall.SocketControlMessage
	msgs, err = syscall.ParseSocketControlMessage(buf)
	if err != nil {
		return 0, err
	}
	fds, err := syscall.ParseUnixRights(&msgs[0])
	if err != nil {
		return 0, err
	}
	return uintptr(fds[0]), nil
}

// ToFile - *os.File from fd
func ToFile(fd uintptr) *os.File {
	return os.NewFile(fd, fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd))
}

// ToConn - net.Conn from fd
func ToConn(fd uintptr) (net.Conn, error) {
	file := ToFile(fd)
	return net.FileConn(file)
}

// ToInode - inode of fd
func ToInode(fd uintptr) (uint64, error) {
	file := ToFile(fd)
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Sys().(*syscall.Stat_t).Ino, nil
}

type fder interface {
	Fd() uintptr
}

type filer interface {
	File() (*os.File, error)
}

// ToFd - fd of a File or net.Conn if possible
func ToFd(thing interface{}) (uintptr, error) {
	if fdthing, ok := thing.(fder); ok {
		return fdthing.Fd(), nil
	}
	if fileThing, ok := thing.(filer); ok {
		file, err := fileThing.File()
		if err != nil {
			return 0, err
		}
		return file.Fd(), nil
	}
	return 0, errors.Errorf("cannot extract fd from %+v", thing)
}
