package gopool

import (
	"errors"
	"io"
	"runtime"
	"sync"
	"time"
)

var (
	ErrParameter = errors.New("parameter error")
	ErrState     = errors.New("pool closed")
)

type Conn interface {
	io.Closer
	Ok(time.Duration) bool
}

type factory func() (Conn, error)

type Pool struct {
	sync.Mutex
	pool      chan Conn
	opts      PoolOptions
	connCount int
	closed    bool
	factory   factory
}

type PoolOptions struct {
	MinConn     int
	MaxConn     int
	IdleTimeout time.Duration
}

type PoolOption func(*PoolOptions)

func newPoolOptions(opt ...PoolOption) PoolOptions {
	opts := PoolOptions{}

	for _, o := range opt {
		o(&opts)
	}

	if opts.MaxConn <= 0 {
		opts.MaxConn = runtime.NumCPU()
	}

	if opts.MinConn > opts.MaxConn {
		opts.MinConn = opts.MaxConn
	}

	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = 0
	}

	return opts
}

func PoolMinConn(n int) PoolOption {
	return func(o *PoolOptions) {
		o.MinConn = n
	}
}

func PoolMaxConn(n int) PoolOption {
	return func(o *PoolOptions) {
		o.MaxConn = n
	}
}

func PoolIdleTimeout(n time.Duration) PoolOption {
	return func(o *PoolOptions) {
		o.IdleTimeout = n
	}
}

func NewPool(factory factory, opt ...PoolOption) (*Pool, error) {
	opts := newPoolOptions(opt...)
	p := &Pool{
		pool:    make(chan Conn, opts.MaxConn),
		opts:    opts,
		closed:  false,
		factory: factory,
	}

	for i := 0; i < opts.MinConn; i++ {
		conn, err := factory()
		if err != nil {
			continue
		}
		p.connCount++
		p.pool <- conn
	}

	return p, nil
}

func (p *Pool) Acquire() (Conn, error) {
	if p.closed {
		return nil, ErrState
	}

TRY_RESOURCE:
	select {
	case conn := <-p.pool:
		if conn.Ok(p.opts.IdleTimeout) {
			return conn, nil
		} else {
			p.Close(conn)
			goto TRY_RESOURCE
		}
	default:
	}

	p.Lock()
	if p.connCount >= p.opts.MaxConn {
		p.Unlock()
		conn := <-p.pool
		if conn.Ok(p.opts.IdleTimeout) {
			return conn, nil
		} else {
			p.Close(conn)
			goto TRY_RESOURCE
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

func (p *Pool) Release(conn Conn) error {
	if p.closed {
		return ErrState
	}

	p.pool <- conn
	return nil
}

func (p *Pool) Close(conn Conn) error {
	err := conn.Close()
	if err != nil {
		return err
	}
	p.Lock()
	p.connCount--
	p.Unlock()
	return nil
}

func (p *Pool) Shutdown() error {
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
