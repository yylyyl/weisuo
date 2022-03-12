package pool

import (
	"errors"
	"log"
	"sync"
	"time"
	"weisuo/protocol"
)

type Pool struct {
	proxy  string
	auth   string
	dialer *protocol.Dialer
	conn   []*protocol.IdleConn
	mutex  sync.RWMutex
	closed bool
}

func MakePool(proxy, auth string, size uint, dialer *protocol.Dialer) *Pool {
	p := &Pool{
		proxy:  proxy,
		auth:   auth,
		dialer: dialer,
		conn:   make([]*protocol.IdleConn, size),
	}
	for i := 0; i < int(size); i++ {
		go p.worker(i)
	}

	return p
}

func (p *Pool) worker(i int) {
	for {
		time.Sleep(500 * time.Millisecond)

		p.mutex.RLock()
		if p.closed {
			p.mutex.RUnlock()
			break
		}
		need := p.conn[i] == nil
		p.mutex.RUnlock()

		if !need {
			continue
		}

		c, err := p.connectIdle(func(cc *protocol.IdleConn) {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			if p.conn[i] == cc {
				log.Printf("ERR POOL remove %s", cc.Id())
				p.conn[i] = nil
			}
		})
		if err != nil {
			log.Printf("ERR POOL connect failure: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		p.mutex.Lock()
		p.conn[i] = c
		p.mutex.Unlock()
		log.Printf("INFO POOL added %s", c.Id())
	}
}

func (p *Pool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.closed = true

	for i, c := range p.conn {
		if c != nil {
			c.Close()
			p.conn[i] = nil
		}
	}
}

func (p *Pool) connectIdle(errCb func(*protocol.IdleConn)) (*protocol.IdleConn, error) {
	return p.dialer.DialIdle(p.proxy, p.auth, errCb)
}

func (p *Pool) Pick() (*protocol.IdleConn, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return nil, errors.New("pool closed")
	}

	var ret *protocol.IdleConn
	for i, c := range p.conn {
		if c != nil {
			ret = c
			p.conn[i] = nil
			break
		}
	}

	return ret, nil
}

func (p *Pool) connectImmediately(proto, target string) (protocol.TCPConn, error) {
	return p.dialer.Dial(p.proxy, p.auth, proto, target)
}

func (p *Pool) Dial(proto, target string) (protocol.TCPConn, error) {
	idleConn, err := p.Pick()
	if err != nil {
		log.Printf("pick err: %v", err)
		return nil, err
	}

	if idleConn != nil {
		return idleConn.Dial(proto, target)
	}

	return p.connectImmediately(proto, target)
}
