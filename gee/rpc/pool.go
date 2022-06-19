package rpc

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/golang/groupcache/singleflight"
)

type myLog struct {
}

func (l *myLog) Info(format string, a ...interface{}) {

}

func (l *myLog) Error(format string, a ...interface{}) {

}

var Log myLog

type ClientPool struct {
	capacity uint64
	addr     string
	next     uint64
	client   []*Client
	sf       *singleflight.Group
}

//connNum<1 默认10个连接
func NewClientPool(addr string, num uint64) *ClientPool {
	if num < 1 {
		num = 2
	}

	pool := &ClientPool{
		addr:     addr,
		capacity: num,
		client:   make([]*Client, num),
		sf:       &singleflight.Group{},
	}

	for i := 0; i < int(num); i++ {
		conn, err := net.Dial("tcp", pool.addr)
		if err != nil {
			Log.Error("dial addr fail,:%v", err)
			continue
		}
		pool.client[i] = NewClient(conn)
		Log.Info("pool add new client:%v", i)
	}
	go pool.loop()
	return pool
}

func (pool *ClientPool) loop() {

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for i := 0; i < int(pool.capacity); i++ {

			idx := i
			if pool.client[idx] != nil && pool.client[idx].IsRunning() {
				continue
			}

			pool.sf.Do(fmt.Sprintf("pool_idx_%d", idx), func() (interface{}, error) {
				conn, err := net.Dial("tcp", pool.addr)
				if err != nil {
					Log.Error("dial addr fail,:%v", err)
					return nil, err
				}

				c := NewClient(conn)
				pool.client[idx] = c
				Log.Info("pool add new client,:%v", idx)
				return c, nil
			})

		}
	}
}

func (pool *ClientPool) GetClient() *Client {
	var (
		idx  uint64
		next uint64
	)
	next = atomic.AddUint64(&pool.next, 1)
	idx = next % pool.capacity

	cli := pool.client[idx]
	if cli != nil {
		if cli.IsRunning() {
			return cli
		}
	}

	tc, err := pool.sf.Do(fmt.Sprintf("pool_idx_%d", idx), func() (interface{}, error) {
		conn, err := net.Dial("tcp", pool.addr)
		if err != nil {
			Log.Error("dial addr fail,:%v", err)
			return nil, err
		}

		c := NewClient(conn)
		pool.client[idx] = c
		Log.Info("pool add new client,:%v", idx)
		return c, nil
	})

	if err != nil && tc != nil {
		cli = tc.(*Client)
	}
	return cli
}
