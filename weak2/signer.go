package main

import (
	"fmt"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in, out := make(chan interface{}), make(chan interface{})
	for _, v := range jobs {

		wg.Add(1)
		go func(in, out chan interface{}, job job) {
			defer wg.Done()
			job(in, out)
			close(out)

		}(in, out, v)

		in = out

	}
	wg.Wait()
}

func SingleHash(in chan interface{}, out chan interface{}) {
	//crc32(data)+"~"+crc32(md5(data))
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	var md, crc, crcWithMd5 string

	v := fmt.Sprint(<-in)
	wg.Add(1)
	go func() {
		defer wg.Done()
		md = DataSignerMd5(v)
	}()

	wg.Add(1)

	mu.Lock()
	go func() {
		defer wg.Done()
		crc = DataSignerCrc32(v)
		mu.Unlock()
	}()
	wg.Add(1)
	mu.Lock()
	go func() {
		defer wg.Done()
		crcWithMd5 = DataSignerCrc32(md)
		mu.Unlock()
	}()
	wg.Wait()

	out <- fmt.Sprintf("%s~%s", crc, crcWithMd5)
	close(out)
}

func MultiHash(in chan interface{}, out chan interface{}) {
	// crc32(th+data)
}

func CombineResults(in chan interface{}, out chan interface{}) {

}

//func workerPool() {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	wg := &sync.WaitGroup{}
//	numbersToProcess, processdNumbers := make(chan int, 5), make(chan int, 5)
//
//	for i := 0; i <= runtime.NumCPU(); i++ {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			worker(ctx, numbersToProcess, processdNumbers)
//		}()
//	}
//
//	go func() {
//		for i := 0; i < 1000; i++ {
//			if i == 500 {
//				cancel()
//			}
//			numbersToProcess <- i
//
//		}
//		close(numbersToProcess)
//	}()
//
//	go func() {
//		wg.Wait()
//		close(processdNumbers)
//	}()
//
//	var counter int
//	for resVal := range processdNumbers {
//		counter++
//		fmt.Println(resVal)
//	}
//
//	fmt.Println(counter)
//
//}
//
//func worker(ctx context.Context, toProcess <-chan int, processed chan<- int) {
//	for {
//		select {
//		case <-ctx.Done():
//			return
//		case value, ok := <-toProcess:
//			if !ok {
//				return
//			}
//			time.Sleep(time.Millisecond * 5)
//			processed <- value * value
//		}
//	}
//}
