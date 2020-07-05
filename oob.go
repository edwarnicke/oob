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
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"syscall"

	"github.com/pkg/errors"
)

// UnixConn - net.UnixConn + SendFD and RecvFD methods for sending and receiving file descriptors
type UnixConn struct {
	*net.UnixConn
}

// New - wrap a *net.UnixConn providing it additional methods to SendFD and RecvFD
func New(s *net.UnixConn) *UnixConn {
	return &UnixConn{s}
}

// SendFD - send the file descriptor fd to the process on the other end of the *net.UnixConn
func (s *UnixConn) SendFD(fd uintptr) error {
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
func (s *UnixConn) RecvFD() (fd uintptr, err error) {
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

// ToInode - inode of fd,*os.File, anything with a method File() (*os.File,error)
func ToInode(thing interface{}) (uint64, error) {
	file, ok := thing.(*os.File)
	if ok {
		fi, err := file.Stat()
		if err != nil {
			return 0, err
		}
		return fi.Sys().(*syscall.Stat_t).Ino, nil
	}
	if fd, ok := thing.(uintptr); ok {
		file = ToFile(fd)
	}
	if f, ok := thing.(filer); ok {
		var err error
		file, err = f.File()
		if err != nil {
			return 0, err
		}
	}
	if file != nil {
		fi, err := file.Stat()
		if err != nil {
			return 0, err
		}
		return fi.Sys().(*syscall.Stat_t).Ino, nil
	}
	return 0, errors.Errorf("Unable to extract an inode for %+v", thing)
}

type fder interface {
	Fd() uintptr
}

type filer interface {
	File() (*os.File, error)
}

// ToFd - fd of a File or net.Conn if possible
func ToFd(thing interface{}) (uintptr, error) {
	if fd, ok := thing.(uintptr); ok {
		return fd, nil
	}
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

// InodeToFile - given an inode, will return n *os.File if and only if the process already has an open fd for that inode
func InodeToFile(inode uint64) *os.File {
	fis, err := ioutil.ReadDir("/proc/self/fd/")
	if err != nil {
		return nil
	}
	for _, fi := range fis {
		// You may be asking yourself... why not just use fi.Sys().(*syscall.Stat_t).Ino, nil
		// The answer is because /proc/self/fd/${fd} is a *link* to the file, with its own distinct Inode
		fd64, err := strconv.ParseUint(fi.Name(), 10, 64)
		if err != nil {
			return nil
		}
		fd := uintptr(fd64)
		fdInode, err := ToInode(fd)
		if err == nil && fdInode == inode {
			return ToFile(fd)
		}
	}
	return nil
}

// InodeToConn - given an inode, will return n net.Conn if and only if the process already has an open fd for that inode and its a connection socketd
func InodeToConn(inode uint64) (net.Conn, error) {
	if file := InodeToFile(inode); file != nil {
		return net.FileConn(file)
	}
	return nil, errors.Errorf("No file found for inode %d", inode)
}
