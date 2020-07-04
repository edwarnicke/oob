oob provides a simple wrapper for [net.UnixConn](https://golang.org/pkg/net/#UnixConn) which
adds two methods:

* ```SendFD(fd uintptr)``` - which sends a file descriptor over the unix file socket and
* ```RecvFD() fd uintptr``` - which receives a file descriptor over the unix file socket

In addition oob provides utility functions:

* ```ToFd(interface{}) (fd uintptr,err error)``` - converts any interface which provides a File() *os.File or Fd() uintptr method and returns its fd.  Handy for *os.File, most implentations of net.Conn, etc
* ```ToFile(fd uintptr) *os.File```- converts an fd to an *os.File with name ```fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)```
* ```ToConn(fd uintptr) (net.Conn,error)``` - converts an fd to a net.Conn
* ```ToInode(fd uintptr) (inode uint64, err error)``` - returns the inode of the fd

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
