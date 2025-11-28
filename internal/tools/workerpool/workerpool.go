package workerpool

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Task func() error
type OnError func(err error)

var ErrQueueFull = errors.New("workerpool: queue full")

const (
	stateInit uint32 = iota
	stateStarted
	stateStopped
)

type WorkerPool struct {
	workerCount int
	tasks       chan Task
	wgWorkers   sync.WaitGroup
	wgTasks     sync.WaitGroup
	onError     OnError
	state       uint32 // 0 - init, 1 - started, 2 - stopped
}

func New(workerCount int, queueSize int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		tasks:       make(chan Task, queueSize),
	}
}

func (wp *WorkerPool) Start() {
	if !atomic.CompareAndSwapUint32(&wp.state, stateInit, stateStarted) {
		return // уже стартовал или остановлен
	}

	for i := 0; i < wp.workerCount; i++ {
		wp.wgWorkers.Add(1)
		go func() {
			defer wp.wgWorkers.Done()
			for task := range wp.tasks {
				func() {
					defer wp.wgTasks.Done()
					if task == nil {
						return
					}
					if err := task(); err != nil && wp.onError != nil {
						wp.onError(err)
					}
				}()
			}
		}()
	}
}

func (wp *WorkerPool) Add(task Task) (err error) {
	state := atomic.LoadUint32(&wp.state)
	if state == stateStopped {
		return errors.New("workerpool: stopped")
	}

	wp.wgTasks.Add(1)

	// Если канал закрыли между проверкой состояния и отправкой — не паниковать.
	defer func() {
		if r := recover(); r != nil {
			// Канал закрыт, снимаем добавленный счетчик и возвращаем ошибку
			wp.wgTasks.Done()
			err = errors.New("workerpool: stopped")
		}
	}()

	// Если канал полон, то сразу вернем ошибку
	select {
	case wp.tasks <- task:
		return nil
	default:
		wp.wgTasks.Done()
		return ErrQueueFull
	}
}

func (wp *WorkerPool) OnError(onError OnError) {
	wp.onError = onError
}

func (wp *WorkerPool) Stop() {
	if !atomic.CompareAndSwapUint32(&wp.state, stateStarted, stateStopped) {
		return // не был запущен или уже остановлен
	}

	close(wp.tasks)

	wp.wgTasks.Wait()
	wp.wgWorkers.Wait()
}
