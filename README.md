# gopool

This is a simple golang tcp connection pool.

## example

```go
type (
	CloseTcpConn func(client interface{}) error
	TcpIsOpen    func(client interface{}) bool
)

type testConn struct {
	conn       interface{}
	connIsOpen TcpIsOpen
	closeConn  CloseTcpConn
	t          time.Time
}

func (tc *testConn) Close() error {
	return tc.closeConn(tc.conn)
}

func (tc *testConn) Good(_ time.Time) bool {
	return true
}

func (tc *testConn) Err() error {
	return nil
}

func (tc *testConn) SetErr(_ error) {
}

func (tc *testConn) Do(_ Action) *Done {
	return &Done{}
}

func (tc *testConn) ForceClose() bool {
	return false
}

func factory() (Conn, error) {
	socket, _ := net.Dial("tcp", "weibo.com:80")
	return &testConn{
		conn: socket,
		connIsOpen: func(conn interface{}) bool {
			return true
		},
		closeConn: func(conn interface{}) error {
			return conn.(*net.TCPConn).Close()
		},
	}, nil
}

func BenchmarkAquire(b *testing.B) {
	pool := NewPool(factory, PoolMaxIdle(30), PoolMaxActive(30))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, _ := pool.Aquire()
		conn.Close()
	}
}

```

## benchmark

```
liubang@venux-dev:~/workspace/go/gopool$ go test -bench=. -run=none
goos: linux
goarch: amd64
pkg: github.com/liubang/gopool
BenchmarkAquire-4   	 4380476	       258 ns/op
PASS
ok  	github.com/liubang/gopool	1.424s
```
