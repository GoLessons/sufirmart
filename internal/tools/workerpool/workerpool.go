package workerpool

import (
	"sync"
)

type Task func() error

type WorkerPool struct {
	workerCount int
	tasks       chan Task
	wgWorkers   sync.WaitGroup
	wgTasks     sync.WaitGroup
	once        sync.Once
	stopCh      chan struct{}
	onError     OnError
}

type OnError func(err error)

func New(workerCount int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		tasks:       make(chan Task, queueSize),
		stopCh:      make(chan struct{}),
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wgWorkers.Add(1)
		go func(id int) {
			defer wp.wgWorkers.Done()
			for {
				select {
				case <-wp.stopCh:
					return
				case task, ok := <-wp.tasks:
					if !ok {
						return
					}
					if task != nil {
						err := task()
						if err != nil && wp.onError != nil {
							wp.onError(err)
						}
						wp.wgTasks.Done()
					}
				}
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
		close(wp.stopCh)
		close(wp.tasks)
		wp.wgTasks.Wait()
		wp.wgWorkers.Wait()
	})
}
