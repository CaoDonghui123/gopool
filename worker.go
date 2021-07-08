package main

import "time"

type Worker struct {
	pool *Pool
	// 删除时间
	removeTime time.Time
	// 任务
	task chan interface{}
	// 到期时间
	expiryTime time.Duration
}

func (w *Worker) run() {
	w.pool.runNum++

	go func() {
		defer func() {
			w.pool.runNum--
			w.pool.workerCache.Put(w)
		}()

		for t := range w.task {
			if nil == t {
				w.pool.runNum--
				w.pool.workerCache.Put(w)
				return
			}
			w.pool.poolFunc(t)
			if ok := w.pool.reWorker(w); !ok {
				break
			}
		}
	}()
}
