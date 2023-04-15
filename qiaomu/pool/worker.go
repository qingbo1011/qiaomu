package pool

import (
	"time"

	qlog "github.com/qingbo1011/qiaomu/log"
)

type Worker struct {
	pool     *Pool
	task     chan func() // 任务队列
	lastTime time.Time   // 执行任务的最后的时间
}

// worker运行
func (w *Worker) run() {
	w.pool.incRunning()
	go w.running()
}

func (w *Worker) running() {
	defer func() {
		w.pool.decRunning()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			// 捕获任务发生的panic
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				qlog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			return
		}
		f()
		w.pool.PutWorker(w) // 任务运行完成，worker空闲，放回协程池
	}
}
