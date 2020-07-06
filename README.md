oob provides a simple wrapper for [net.UnixConn](https://golang.org/pkg/net/#UnixConn) which
adds two methods:

* ```SendFD(fd uintptr)``` - which sends a file descriptor over the unix file socket and
* ```RecvFD() fd uintptr``` - which receives a file descriptor over the unix file socket

In addition oob provides utility functions:

* ```ToFd(interface{}) (fd uintptr,err error)``` - converts anything which provides the SyscallConn() (syscall.RawConn, error) or inode its fd.
* ```ToFile(interface{}) *os.File```- converts anything which provides the SyscallConn() (syscall.RawConn, error),fd, or inode its to an *os.File with name ```fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)```
* ```ToConn(interface{}) (net.Conn,error)``` - converts anything which provides the SyscallConn() (syscall.RawConn, error)fd, or inode its to a net.Conn
* ```ToInode(interface{}) (inode uint64, err error)``` - converts anything which provides the SyscallConn() (syscall.RawConn, error) or fd to it inode

# Compatibility and Dockerfile
oob only works on linux.

oob is a go library, not an executable.  A Dockerfile is provided to aid those doing dev in
other environments.

```dockerfile
docker run $(docker build -q . --target test)
```

will run tests.

```dockerfile
docker run -p 40000:40000 $(docker build -q . --target debug)
```

will run tests with a debugger listening on port 40000
