package main

import (
	"fmt"
	"strings"
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
	input := make(map[int]string)
	for data := range in {
		if receivedMap, ok := data.(map[int]string); ok {
			for key, value := range receivedMap {
				input[key] = value
			}
		}
	}
	th := 6
	myRes := make(map[int][]string)
	for key, _ := range input {
		myRes[key] = make([]string, th)
	}
	wg := &sync.WaitGroup{}
	for n, _ := range myRes {
		wg.Add(1)
		go func(n int, wg *sync.WaitGroup) {
			defer wg.Done()
			wgInner := &sync.WaitGroup{}
			for i := 0; i < th; i++ {
				wgInner.Add(1)
				go func(n int, prefix int, wgInner *sync.WaitGroup) {
					defer wgInner.Done()
					gg := fmt.Sprintf("%v%v", prefix, input[n])
					hashRes := DataSignerCrc32(gg)
					myRes[n][prefix] = hashRes
				}(n, i, wgInner)
			}
			wgInner.Wait()
			out <- map[int][]string{n: myRes[n]}
		}(n, wg)
	}
	wg.Wait()
	close(out)
}

func CombineResults(in chan interface{}, out chan interface{}) {
	res := make(map[int][]string)
	counter := 0
	for val := range in {
		m, ok := val.(map[int][]string)
		if !ok {
			continue
		}
		for k, v := range m {
			res[k] = v
		}
		counter++
	}
	th := 6
	cnt := 0
	var builder strings.Builder
	builder.Grow(counter * (th + 1))
	for i := 0; i < counter; i++ {
		for _, v := range res[i] {
			builder.WriteString(v)
			cnt++
			if cnt%th == 0 {
				if i+1 != counter {
					builder.WriteString("-")
				}
			}
		}
	}
	result := builder.String()
	out <- result
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
