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
			close(out)
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
	for key := range input {
		myRes[key] = make([]string, th)
	}
	wg := &sync.WaitGroup{}
	for n := range myRes {
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
					builder.WriteString("_")
				}
			}
		}
	}
	result := builder.String()
	out <- result
}
