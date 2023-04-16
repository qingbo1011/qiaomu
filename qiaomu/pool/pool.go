package pool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qingbo1011/qiaomu/config"
)

type sig struct{}

const DefaultExpireTime = 5

var (
	ErrorInValidCap    = errors.New("pool cap can not <= 0")
	ErrorInValidExpire = errors.New("pool expire can not <= 0")
	ErrorHasClosed     = errors.New("pool has bean released")
)

type Pool struct {
	cap          int32         // 容量 (类似切片的cap,协程池最大容量 pool max cap)
	running      int32         // 正在运行的worker的数量
	workers      []*Worker     // 空闲worker
	expire       time.Duration // 过期时间(空闲的worker超过这个时间就会被回收)
	release      chan sig      // 释放资源(释放后协程池pool就不能使用了)
	lock         sync.Mutex    // 通过互斥锁来保护pool里面的相关资源的安全
	once         sync.Once     // 保证释放操作release只能调用一次，不能多次调用
	workerCache  sync.Pool     // 缓存(worker的创建可以放在缓存中，提升效率)
	cond         *sync.Cond    // 基于互斥锁实现的条件变量，用来协调想要访问共享资源的goroutine
	PanicHandler func()        // 协程池异常处理
}

// NewPool 创建协程池(默认过期时间5s)
func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpireTime)
}

// NewTimePool 创建协程池(默认过期由用户指定)
func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInValidCap
	}
	if expire <= 0 {
		return nil, ErrorInValidExpire
	}
	p := &Pool{
		cap:     int32(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	p.cond = sync.NewCond(&p.lock)

	go p.expireWorker()
	return p, nil
}

// NewPoolConf 根据配置文件创建协程池
func NewPoolConf() (*Pool, error) {
	cap, ok := config.Conf.Pool["cap"]
	if !ok {
		return nil, errors.New("cap config not exist")
	}
	return NewTimePool(int(cap.(int64)), DefaultExpireTime)
}

// 定时清理过期的空闲worker
func (p *Pool) expireWorker() {
	ticker := time.NewTicker(p.expire) // 利用time.NewTicker实现定时器
	for range ticker.C {
		if p.IsClosed() {
			break
		}
		// 遍历空闲的worker：如果当前时间和worker的最后运行任务的时间的差值大于expire，则对该worker进行清理
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n >= 0 {
			var clearN = -1
			for i, w := range idleWorkers {
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				clearN = i
				w.task <- nil
				idleWorkers[i] = nil
			}
			if clearN != -1 {
				if clearN >= len(idleWorkers)-1 {
					p.workers = idleWorkers[:0]
				} else {
					p.workers = idleWorkers[clearN+1:]
				}
				fmt.Printf("清除完成,running:%d, workers:%v \n", p.running, p.workers)
			}
		}
		p.lock.Unlock()
	}
}

// Submit 给协程池提交任务
func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	// 获取协程池里面的一个worker，然后执行任务
	w := p.GetWorker()
	w.task <- task
	return nil
}

// GetWorker 获取协程池中的空闲worker
func (p *Pool) GetWorker() (w *Worker) {
	// 如果有空闲的worker,直接获取
	readyWorker := func() {
		w = p.workerCache.Get().(*Worker)
		w.run()
	}
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w = idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return
	}
	// 如果没有空闲的worker，而运行中的 worker 数量小于协程池 pool 的容量，就新建一个worker；
	if p.running < p.cap {
		p.lock.Unlock()
		readyWorker()
		return
	}
	p.lock.Unlock()
	// 如果正在运行的worker数量大于pool容量，则阻塞等待，直到有worker释放
	return p.waitIdleWorker()
}

// 获取worker(如果没有空闲worker则等待)
func (p *Pool) waitIdleWorker() *Worker {
	p.lock.Lock()
	p.cond.Wait()

	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		p.lock.Unlock()
		if p.running < p.cap { // 正在运行的worker数量小于pool的容量，才可以新建worker
			c := p.workerCache.Get()
			var w *Worker
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				w = c.(*Worker)
			}
			w.run()
			return w
		}
		return p.waitIdleWorker()
	}
	w := idleWorkers[n]
	idleWorkers[n] = nil
	p.workers = idleWorkers[:n]
	p.lock.Unlock()
	return w
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

// PutWorker 将指定worker回收到协程池中
func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.cond.Signal()
	p.lock.Unlock()
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

// Release 释放协程池
func (p *Pool) Release() {
	p.once.Do(func() {
		//只执行一次
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			if w == nil {
				continue
			}
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

// IsClosed 判断协程池是否关闭
func (p *Pool) IsClosed() bool {
	return len(p.release) > 0
}

// Restart 重启协程池
func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	go p.expireWorker()
	return true
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free 返回协程池中空闲的worker数量
func (p *Pool) Free() int {
	return int(p.cap - p.running)
}
