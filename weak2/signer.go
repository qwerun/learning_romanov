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

		}(in, out, v)

		in = out

	}
	wg.Wait()
}

func SingleHash(in chan interface{}, out chan interface{}) {
	//crc32(data)+"~"+crc32(md5(data))
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	counter := -1
	for val := range in {
		counter++
		v := fmt.Sprint(val)
		wg.Add(1)
		go func(v string, num int) {
			defer wg.Done()

			md5Ch, crc32Ch := make(chan string), make(chan string)

			go func() {
				mu.Lock()
				md5Ch <- DataSignerMd5(v)
				mu.Unlock()
				close(md5Ch)
			}()

			go func() {
				crc32Ch <- DataSignerCrc32(v)
				close(crc32Ch)
			}()

			md5Hash := <-md5Ch
			crcWithMd5 := DataSignerCrc32(md5Hash)
			res := map[int]string{num: fmt.Sprintf("%s~%s", <-crc32Ch, crcWithMd5)}
			out <- res
		}(v, counter)
	}
	wg.Wait()
	close(out)
}

func MultiHash(in chan interface{}, out chan interface{}) {
	// crc32(th+data)
	input := make(map[int]string)
	for data := range in {
		if receivedMap, ok := data.(map[int]string); ok {
			for key, value := range receivedMap {
				input[key] = value
			}
		}
	}

	th := 6
	res := make(map[int]map[string][]string)

	for key, inVal := range input {
		res[key] = make(map[string][]string)
		res[key][inVal] = make([]string, th)
	}

	wg := &sync.WaitGroup{}
	for n, val := range res {
		for key := range val {
			wg.Add(1)
			go func(key string, n int, wg *sync.WaitGroup) {
				defer wg.Done()
				wgInner := &sync.WaitGroup{}
				for i := 0; i < th; i++ {
					wgInner.Add(1)
					go func(v string, num int, keyMap int, wg *sync.WaitGroup) {
						defer wgInner.Done()
						gg := fmt.Sprintf("%v%v", num, v)
						hashRes := DataSignerCrc32(gg)
						res[keyMap][key][i] = hashRes
					}(key, i, n, wgInner)
				}
				wgInner.Wait()
				out <- map[int]map[string][]string{n: val}
			}(key, n, wg)
		}
	}

	wg.Wait()
	close(out)
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
