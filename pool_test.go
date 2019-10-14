package gopool

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

type TcpConn struct {
	conn net.Conn
	t    time.Time
}

func (tp *TcpConn) Close() error {
	return tp.conn.Close()
}

func (tp *TcpConn) Ok(idleTimeout time.Duration) bool {
	return tp.t.Add(idleTimeout).After(time.Now())
}

func TestCommonPool(t *testing.T) {
	factory := func() (Conn, error) {

		c, e := net.Dial("tcp", "venux-dev:80")
		if e != nil {
			return nil, e
		}
		return &TcpConn{
			conn: c,
			t:    time.Now(),
		}, nil
	}
	pool, err := NewPool(factory, PoolMinConn(5), PoolMaxConn(10), PoolIdleTimeout(3*time.Second))
	if err != nil {
		t.Error(err)
	}

	defer pool.Shutdown()
	wg := sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func(pool *Pool, i int) {
			c, e := pool.Acquire()
			if e != nil {
				t.Error(e)
			}
			fmt.Println(i, "do sth.")
			time.Sleep(time.Second * 4)
			e = pool.Release(c)
			if e != nil {
				t.Error(e)
			}
			fmt.Println(i, "done.")
			wg.Done()
		}(pool, i)
	}

	wg.Wait()
}
