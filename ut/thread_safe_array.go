package ut

import "sync"

// ThreadSafeArray 线程安全的数组
type ThreadSafeArray[T any] struct {
	array []T
	lock  sync.RWMutex
}

func (t *ThreadSafeArray[T]) Append(value ...T) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.array = append(t.array, value...)
}

func (t *ThreadSafeArray[T]) Get(index int) T {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.array[index]
}

func (t *ThreadSafeArray[T]) GetAll() []T {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.array
}
func (t *ThreadSafeArray[T]) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.array)
}
func (t *ThreadSafeArray[T]) Pop(count int) []T {
	t.lock.Lock()
	defer t.lock.Unlock()

	if count > len(t.array) {
		count = len(t.array)
	}

	result := t.array[len(t.array)-count:]
	t.array = t.array[:len(t.array)-count]

	return result
}
