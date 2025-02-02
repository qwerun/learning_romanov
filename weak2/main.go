package main

import (
	"fmt"
	"sync"
)

//func main() {
//	//in := []int{0, 1}
//	in := make(chan int)
//	go func(in chan int) {
//		defer close(in)
//		aboba := []int{0, 1}
//		for y := range aboba {
//			in <- y
//		}
//
//	}(in)
//	wg := &sync.WaitGroup{}
//	mu := &sync.Mutex{}
//	counter := -1
//	for val := range in {
//		mu.Lock()
//		counter++
//		mu.Unlock()
//		v := fmt.Sprint(val)
//		wg.Add(1)
//		go func(v string, num int) {
//			defer wg.Done()
//
//			md5Ch, crc32Ch := make(chan string), make(chan string)
//
//			go func() {
//				mu.Lock()
//				md5Ch <- DataSignerMd5(v)
//				mu.Unlock()
//				close(md5Ch)
//			}()
//
//			go func() {
//				crc32Ch <- DataSignerCrc32(v)
//				close(crc32Ch)
//			}()
//
//			md5Hash := <-md5Ch
//			crcWithMd5 := DataSignerCrc32(md5Hash)
//			res := map[int]string{num: fmt.Sprintf("%s~%s", <-crc32Ch, crcWithMd5)}
//			fmt.Println(res)
//
//		}(v, counter)
//
//	}
//
//	wg.Wait()
//}

func main() {
	in := map[int]string{
		0: "4108050209~502633748",
		1: "2212294583~709660146",
	}
	//in := make(chan int)
	//go func(in chan int) {
	//	defer close(in)
	//	aboba := map[int]string{
	//		0: "4108050209~502633748",
	//		1: "2212294583~709660146",
	//	}
	//	for y := range aboba {
	//		in <- y
	//	}
	//}(in)
	th := 6
	res := make(map[int]map[string][]string)

	for key, inVal := range in {
		res[key] = make(map[string][]string)
		res[key][inVal] = make([]string, th)
	}

	wg := &sync.WaitGroup{}
	for n, val := range res {

		for key := range val {
			for i := 0; i < th; i++ {
				wg.Add(1)
				go func(v string, num int, wg *sync.WaitGroup) {
					defer wg.Done()
					gg := fmt.Sprintf("%v%v", num, v)
					hashRes := DataSignerCrc32(gg)

					res[n][key][i] = hashRes

				}(key, i, wg)

			}
		}

	}

	wg.Wait()
	fmt.Println(res)

}
