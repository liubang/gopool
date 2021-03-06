package gopool

import (
	"net"
	"testing"
	"time"
)

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

func f() (Conn, error) {
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
	pool := NewPool(f, PoolMaxIdle(30), PoolMaxActive(30))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, _ := pool.Aquire()
		conn.Close()
	}
}
