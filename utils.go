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

// ToFile - *os.File from fd
func ToFile(thing interface{}) (*os.File, error) {
	// Is it a file?
	if file, ok := thing.(*os.File); ok {
		return file, nil
	}

	// Can I get a fd from it?
	if fd, err := ToFd(thing); err == nil {
		return os.NewFile(fd, fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)), nil
	}
	return nil, errors.Errorf("cannot create *os.File for %+v", thing)
}

// ToConn - net.Conn from fd
func ToConn(thing interface{}) (net.Conn, error) {
	if conn, ok := thing.(net.Conn); ok {
		return conn, nil
	}
	file, err := ToFile(thing)
	if err != nil {
		return nil, err
	}
	conn, err := net.FileConn(file)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return conn, nil
}

type syscallconner interface {
	SyscallConn() (syscall.RawConn, error)
}

// ToFd - fd of a File or net.Conn if possible
func ToFd(thing interface{}) (uintptr, error) {
	// Is it a uintptr (ie, a fd)
	if fd, ok := thing.(uintptr); ok {
		// Is it really an fd?
		if file := os.NewFile(fd, ""); file == nil {
			return 0, errors.Errorf("fd %d is not a valid file descriptor", fd)
		}
		return fd, nil
	}

	// Is it a uint64 (ie, an inode)
	if inode, ok := thing.(uint64); ok {
		fis, err := ioutil.ReadDir("/proc/self/fd/")
		if err != nil {
			return 0, err
		}
		for _, fi := range fis {
			// You may be asking yourself... why not just use fi.Sys().(*syscall.Stat_t).Ino, nil
			// The answer is because /proc/self/fd/${fd} is a *link* to the file, with its own distinct Inode
			fd64, err := strconv.ParseUint(fi.Name(), 10, 64)
			if err != nil {
				return 0, errors.WithStack(err)
			}
			fd := uintptr(fd64)
			fdInode, err := ToInode(fd)
			if err == nil && fdInode == inode {
				return fd, nil
			}
		}
		return 0, errors.Errorf("cannot find fd in /proc/%d/fd/* for inode %d", os.Getpid(), inode)
	}

	// Does it provide a syscall.RawCall?
	if scc, ok := thing.(syscallconner); ok {
		rawconn, err := scc.SyscallConn()
		if err != nil {
			return 0, err
		}
		fdchan := make(chan uintptr, 1)
		err = rawconn.Control(func(fd uintptr) {
			fdchan <- fd
			close(fdchan)
		})
		if err != nil {
			return 0, err
		}
		return <-fdchan, nil
	}

	return 0, errors.Errorf("cannot extract fd from %+v", thing)
}

// ToInode - inode of fd,*os.File, anything with a method File() (*os.File,error)
func ToInode(thing interface{}) (uint64, error) {
	// Is it already a uint64 and thus presumably an inode?
	if inode, ok := thing.(uint64); ok {
		// Is it *really* an inode though?
		fis, err := ioutil.ReadDir("/proc/self/fd/")
		if err != nil {
			return 0, err
		}
		for _, fi := range fis {
			// You may be asking yourself... why not just use fi.Sys().(*syscall.Stat_t).Ino, nil
			// The answer is because /proc/self/fd/${fd} is a *link* to the file, with its own distinct Inode
			fd64, err := strconv.ParseUint(fi.Name(), 10, 64)
			if err != nil {
				return 0, errors.WithStack(err)
			}
			fd := uintptr(fd64)
			fdInode, err := ToInode(fd)
			if err == nil && fdInode == inode {
				return inode, nil
			}
		}
		return inode, nil
	}

	file, err := ToFile(thing)
	if err != nil {
		return 0, err
	}
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Sys().(*syscall.Stat_t).Ino, nil
}
