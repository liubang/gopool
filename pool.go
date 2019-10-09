package gopool

import (
	"errors"
	"io"
	"sync"
)

var (
	ErrParameter = errors.New("parameter error")
	ErrState     = errors.New("pool closed")
)

type Conn interface {
	io.Closer
	Alive() bool
}

type factory func() (Conn, error)

type Pool interface {
	Acquire() (Conn, error)
	Release(Conn) error
	Close(Conn) error
	Shutdown() error
}

type CommonPool struct {
	sync.Mutex
	pool      chan Conn
	minConn   int
	maxConn   int
	connCount int
	closed    bool
	factory   factory
}

func NewCommonPool(minConn, maxConn int, factory factory) (*CommonPool, error) {
	if maxConn < minConn {
		return nil, ErrParameter
	}

	if maxConn <= 0 {
		// TODO use cpu number
		maxConn = 4
	}

	p := &CommonPool{
		pool:    make(chan Conn, maxConn),
		maxConn: maxConn,
		closed:  false,
		factory: factory,
	}

	for i := 0; i < minConn; i++ {
		conn, err := factory()
		if err != nil {
			continue
		}
		p.connCount++
		p.pool <- conn
	}

	return p, nil
}

func (p *CommonPool) Acquire() (Conn, error) {
	if p.closed {
		return nil, ErrState
	}

TRY:
	select {
	case conn := <-p.pool:
		if conn.Alive() {
			return conn, nil
		} else {
			conn.Close()
			p.Lock()
			p.connCount--
			p.Unlock()
			goto TRY
		}
	default:
	}

	p.Lock()
	if p.connCount >= p.maxConn {
		p.Unlock()
		conn := <-p.pool
		if conn.Alive() {
			return conn, nil
		} else {
			conn.Close()
			p.Lock()
			p.connCount--
			p.Unlock()
			goto TRY
		}
	} else {
		conn, err := p.factory()
		if err != nil {
			p.Unlock()
			return nil, err
		}
		p.connCount++
		p.Unlock()
		return conn, nil
	}
}

func (p *CommonPool) Release(conn Conn) error {
	if p.closed {
		return ErrState
	}

	p.pool <- conn
	return nil
}

func (p *CommonPool) Close(conn Conn) error {
	err := conn.Close()
	if err != nil {
		return err
	}
	p.Lock()
	p.connCount--
	p.Unlock()
	return nil
}

func (p *CommonPool) Shutdown() error {
	if p.closed {
		return nil
	}

	p.Lock()
	close(p.pool)
	for conn := range p.pool {
		conn.Close()
		p.connCount--
	}
	p.closed = true
	p.Unlock()
	return nil
}
