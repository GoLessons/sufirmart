package workerpool

import (
	"sync"
	"sync/atomic"
)

type Task func() error

type WorkerPool struct {
	workerCount int
	tasks       chan Task
	wgWorkers   sync.WaitGroup
	wgTasks     sync.WaitGroup
	once        sync.Once
	onError     OnError
	started     uint32 // 0 - not started, 1 - started
}

type OnError func(err error)

func New(workerCount int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		tasks:       make(chan Task, queueSize),
	}
}

func (wp *WorkerPool) Start() {
	if atomic.LoadUint32(&wp.started) == 1 {
		return
	}

	atomic.StoreUint32(&wp.started, 1)
	for i := 0; i < wp.workerCount; i++ {
		wp.wgWorkers.Add(1)
		go func(id int) {
			defer wp.wgWorkers.Done()
			for task := range wp.tasks {
				if task != nil {
					if err := task(); err != nil && wp.onError != nil {
						wp.onError(err)
					}
				}
				wp.wgTasks.Done()
			}
		}(i)
	}
}

func (wp *WorkerPool) Add(task Task) {
	wp.wgTasks.Add(1)
	wp.tasks <- task
}

func (wp *WorkerPool) OnError(onError OnError) {
	wp.onError = onError
}

func (wp *WorkerPool) Stop() {
	wp.once.Do(func() {
		close(wp.tasks)

		if atomic.LoadUint32(&wp.started) == 1 {
			wp.wgTasks.Wait()
			wp.wgWorkers.Wait()
			return
		}

		for range wp.tasks {
			wp.wgTasks.Done()
		}
	})
}
