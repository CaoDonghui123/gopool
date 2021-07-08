package main

import (
	"errors"
	"runtime"
	"sync"
	"time"
)

const (
	CLEANTIME = 1
	CLOSE     = 1
)

type Pool struct {
	//容量
	capacity int
	//运行数
	runNum int

	workers []*Worker

	cleanTime time.Duration
	//
	poolFunc func(interface{})

	lock sync.Mutex
	//关闭flag
	release int

	workerCache sync.Pool

	once sync.Once

	cond *sync.Cond
}

func NewPool(size int, pf func(interface{})) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("size need > 0")
	}
	p := &Pool{
		capacity:  size,
		cleanTime: time.Duration(CLEANTIME) * time.Second,
		poolFunc:  pf,
	}
	p.cond =sync.NewCond(&p.lock)
	go p.periodicallyClean()
	return p, nil
}

// 定期清除
func (p *Pool) periodicallyClean() {
	// NewTicker是周期性定时器
	// 每过一段时间，会往chan中发送时间
	heartbeat := time.NewTicker(p.cleanTime)
	defer heartbeat.Stop()
	for range heartbeat.C {
		if CLOSE == p.release {
			break
		}
		now := time.Now()
		p.lock.Lock()
		workers := p.workers
		n := -1
		for i, w := range workers {
			if now.Sub(w.removeTime) <= p.cleanTime {
				break
			}
			n = i
			w.task <- nil
			workers[i] = nil
		}
		if n > -1 {
			if n >= len(workers)-1 {
				p.workers = workers[:0]
			} else {
				p.workers = workers[n+1:]
			}
		}
		p.lock.Unlock()
	}
}

func (p *Pool) Submit(task interface{}) error {
	if CLOSE == p.release {
		return errors.New("closed")
	}
	p.retrieveWorker().task <- task
	return nil
}

func (p *Pool) retrieveWorker() *Worker {
	var w *Worker
	p.lock.Lock()
	workers := p.workers
	n := len(workers) - 1
	if n >= 0 {
		w = workers[n]
		workers[n] = nil
		p.workers = workers[:n]
		p.lock.Unlock()
	} else if p.runNum < p.capacity {
		p.lock.Unlock()
		if cache := p.workerCache.Get(); cache != nil {
			w = cache.(*Worker)
		} else {
			w = &Worker{
				pool: p,
				task: make(chan interface{}, func() int {
					if runtime.GOMAXPROCS(0) == 1 {
						return 0
					}
					return 1
				}()),
			}
		}
		w.run()
	} else {
		for {
			p.cond.Wait()
			l := len(p.workers) - 1
			if l < 0 {
				continue
			}
			w = p.workers[l]
			p.workers[l] = nil
			p.workers = p.workers[:l]
			break
		}
		p.lock.Unlock()

	}
	return w
}

func (p *Pool) reWorker(worker *Worker) bool {
	if CLOSE == p.release {
		return false
	}
	worker.removeTime = time.Now()
	p.lock.Lock()
	//defer p.lock.Unlock()
	p.workers = append(p.workers, worker)
	p.cond.Signal()
	p.lock.Unlock()
	return true
}

func (p *Pool) Close() error {
	p.once.Do(func() {
		p.release = 1
		p.lock.Lock()
		//defer p.lock.Unlock()
		workers := p.workers
		for i, w := range workers {
			w.task <- nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
	})
	return nil
}
