package main

import (
	"fmt"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, v := range jobs {

		out := make(chan interface{})
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
	for val := range in {
		v := fmt.Sprint(val)
		wg.Add(1)
		go func(v string) {
			defer wg.Done()
			md5Ch, crc32Ch := make(chan string), make(chan string)

			go func() {
				md5Ch <- DataSignerMd5(v)
				close(md5Ch)
			}()

			go func() {
				mu.Lock()
				crc32Ch <- DataSignerCrc32(v)
				mu.Unlock()
				close(crc32Ch)
			}()

			md5Hash := <-md5Ch
			mu.Lock()
			crcWithMd5 := DataSignerCrc32(md5Hash)
			mu.Unlock()
			out <- fmt.Sprintf("%s~%s", <-crc32Ch, crcWithMd5)

		}(v)
	}

	wg.Wait()

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
