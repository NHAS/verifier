package threadmgmt

import (
	"errors"
	"sync"
)

type ProducerFunc func(work chan<- interface{}) error
type WorkerFunc func(interface{}) (key string, value string, err error)
type CollectionCounterFunc func(key string, val string)
type ResultFunc func(map[string]string)

var BadType = errors.New("Unknown type passed to thread consumer")
var Unrecoverable = errors.New("An unrecoverable error was encountered by a consumer thread")
var SkipCollecting = errors.New("Signal for thread manager to not store value. Not an error.")
var CountButSkipCollecting = errors.New("Signal for thread manager to not store value but still count record. Not an error.")

type pair struct {
	key   string
	value string
	err   error
}

func Start(p ProducerFunc, w WorkerFunc, c CollectionCounterFunc, r ResultFunc, numConsumers int) error {
	var tp threadpool
	tp.producer = p
	tp.worker = w
	tp.collectionCounter = c
	tp.result = r
	return tp.poolManager(numConsumers)
}

type threadpool struct {
	wg                sync.WaitGroup
	producer          ProducerFunc
	worker            WorkerFunc
	result            ResultFunc
	collectionCounter CollectionCounterFunc
}

func (tp threadpool) poolManager(consumers int) (err error) {
	results := make(chan pair)
	work := make(chan interface{})
	errc := make(chan error, 1)

	tp.wg.Add(consumers)

	for i := 0; i < consumers; i++ {
		go func() {
			tp.consumer(work, results)

			tp.wg.Done()
		}()
	}

	go func() {
		defer close(work)
		errc <- tp.producer(work)
	}()

	go func() {
		tp.wg.Wait()
		close(results)
	}()

	tp.result(tp.collect(results))

	return nil // Change this
}

func (tp threadpool) collect(results <-chan pair) (collection map[string]string) {
	collection = make(map[string]string)

	for result := range results {
		if result.err == BadType || result.err == Unrecoverable {
			panic(result.err)
		}

		if result.err != SkipCollecting && result.err != CountButSkipCollecting {
			collection[result.key] = result.value
		}

		if result.err != SkipCollecting {
			tp.collectionCounter(result.key, result.value)
		}

	}
	return collection
}

func (tp threadpool) consumer(workItems <-chan interface{}, result chan<- pair) {
	for item := range workItems {

		key, val, err := tp.worker(item)
		result <- pair{key, val, err}
	}
}
