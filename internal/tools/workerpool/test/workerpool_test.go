package test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sufirmart/internal/tools/workerpool"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool_ExecutesAllTasks(t *testing.T) {
	wp := workerpool.New(5, 100)
	wp.Start()
	defer wp.Stop()

	const tasksCount = 50
	var mu sync.Mutex
	completed := 0

	var wg sync.WaitGroup
	wg.Add(tasksCount)

	for i := 0; i < tasksCount; i++ {
		wp.Add(func() error {
			defer wg.Done()
			mu.Lock()
			completed++
			mu.Unlock()
			return nil
		})
	}

	wg.Wait()
	wp.Stop()

	assert.Equal(t, tasksCount, completed, "Должны выполниться все задачи")
}

func TestWorkerPool_OnErrorCalledForEachError(t *testing.T) {
	wp := workerpool.New(3, 20)

	var mu sync.Mutex
	cbCount := 0
	wp.OnError(func(err error) {
		mu.Lock()
		cbCount++
		mu.Unlock()
	})

	wp.Start()
	defer wp.Stop()

	const total = 10
	const errorTasks = 6

	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i < total; i++ {
		if i < errorTasks {
			wp.Add(func() error {
				defer wg.Done()
				return errors.New("expected error")
			})
		} else {
			wp.Add(func() error {
				defer wg.Done()
				return nil
			})
		}
	}

	wg.Wait()
	wp.Stop()

	assert.Equal(t, errorTasks, cbCount)
}

func TestWorkerPool_ConcurrencyLevel(t *testing.T) {
	const workers = 2
	wp := workerpool.New(workers, 10)
	wp.Start()
	defer wp.Stop()

	const tasksCount = 6
	var wg sync.WaitGroup
	wg.Add(tasksCount)

	var mu sync.Mutex
	running := 0
	maxRunning := 0

	task := func() error {
		mu.Lock()
		running++
		if running > maxRunning {
			maxRunning = running
		}
		mu.Unlock()

		time.Sleep(120 * time.Millisecond)

		mu.Lock()
		running--
		mu.Unlock()

		wg.Done()
		return nil
	}

	for i := 0; i < tasksCount; i++ {
		wp.Add(task)
	}

	wg.Wait()
	wp.Stop()

	assert.Equal(t, workers, maxRunning)
}

func TestWorkerPool_StopIsIdempotent(t *testing.T) {
	wp := workerpool.New(1, 1)
	wp.Start()

	var wg sync.WaitGroup
	wg.Add(1)

	wp.Add(func() error {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	wg.Wait()

	assert.NotPanics(t, func() {
		wp.Stop()
		wp.Stop()
	}, "Повторный вызов Stop не должен паниковать")
}

func TestWorkerPool_ErrorWithoutOnErrorDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		wp := workerpool.New(1, 1)
		wp.Start()

		var wg sync.WaitGroup
		wg.Add(1)

		wp.Add(func() error {
			defer wg.Done()
			return errors.New("any error")
		})

		wg.Wait()
		wp.Stop()
	}, "Возврат ошибки из задачи без установленного OnError не должен приводить к панике")
}
