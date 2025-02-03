package main

//
//
//import (
//	"fmt"
//	"strings"
//	"time"
//)
//
//func main() {
//	in := make(chan interface{})
//
//	go func(in chan interface{}) {
//		//map[0:[2956866606 803518384 1425683795 3407918797 2730963093 1025356555]]
//		//map[1:[495804419 2186797981 4182335870 1720967904 259286200 2427381542]]
//		defer close(in)
//		in <- map[int][]string{0: {"2956866606", "803518384", "1425683795", "3407918797", "2730963093", "1025356555"}}
//		time.Sleep(time.Second)
//		in <- map[int][]string{1: {"495804419", "2186797981", "4182335870", "1720967904", "259286200", "2427381542"}}
//
//	}(in)
//
//	res := make(map[int][]string)
//	counter := 0
//	for val := range in {
//		m, ok := val.(map[int][]string)
//		if !ok {
//			continue
//		}
//		for k, v := range m {
//			res[k] = v
//		}
//		counter++
//	}
//	th := 6
//	cnt := 0
//	var builder strings.Builder
//	builder.Grow(counter * (th + 1))
//	for i := 0; i < counter; i++ {
//		for _, v := range res[i] {
//			builder.WriteString(v)
//			cnt++
//			if cnt%th == 0 {
//				if i+1 != counter {
//					builder.WriteString("-")
//				}
//			}
//		}
//	}
//	result := builder.String()
//	//out <- result
//	fmt.Println(result)
//}
//
////func main() {
//// //func SingleHash
////	//in := []int{0, 1}
////	in := make(chan int)
////	go func(in chan int) {
////		defer close(in)
////		aboba := []int{0, 1}
////		for y := range aboba {
////			in <- y
////		}
////
////	}(in)
////	wg := &sync.WaitGroup{}
////	mu := &sync.Mutex{}
////	counter := -1
////	for val := range in {
////		mu.Lock()
////		counter++
////		mu.Unlock()
////		v := fmt.Sprint(val)
////		wg.Add(1)
////		go func(v string, num int) {
////			defer wg.Done()
////
////			md5Ch, crc32Ch := make(chan string), make(chan string)
////
////			go func() {
////				mu.Lock()
////				md5Ch <- DataSignerMd5(v)
////				mu.Unlock()
////				close(md5Ch)
////			}()
////
////			go func() {
////				crc32Ch <- DataSignerCrc32(v)
////				close(crc32Ch)
////			}()
////
////			md5Hash := <-md5Ch
////			crcWithMd5 := DataSignerCrc32(md5Hash)
////			res := map[int]string{num: fmt.Sprintf("%s~%s", <-crc32Ch, crcWithMd5)}
////			fmt.Println(res)
////
////		}(v, counter)
////
////	}
////
////	wg.Wait()
////}
//
////func main() {
////	// func MultiHash
////	in := make(chan interface{})
////
////	go func(in chan interface{}) {
////		defer close(in)
////		aboba := map[int]string{
////			0: "4108050209~502633748",
////			1: "2212294583~709660146",
////		}
////		in <- aboba
////	}(in)
////
////	input := make(map[int]string)
////	for data := range in {
////		if receivedMap, ok := data.(map[int]string); ok {
////			for key, value := range receivedMap {
////				input[key] = value
////			}
////		}
////	}
////	//fmt.Println(input)
////	//input := map[int]string{
////	//	0: "4108050209~502633748",
////	//	1: "2212294583~709660146",
////	//}
////
////	th := 6
////	myRes := make(map[int][]string)
////
////	for key, _ := range input {
////		myRes[key] = make([]string, th)
////	}
////
////	wg := &sync.WaitGroup{}
////	for n, _ := range myRes {
////		wg.Add(1)
////		go func(n int, wg *sync.WaitGroup) {
////			defer wg.Done()
////			wgInner := &sync.WaitGroup{}
////			for i := 0; i < th; i++ {
////				wgInner.Add(1)
////				go func(n int, prefix int, wgInner *sync.WaitGroup) {
////					defer wgInner.Done()
////					gg := fmt.Sprintf("%v%v", prefix, input[n])
////					hashRes := DataSignerCrc32(gg)
////					myRes[n][prefix] = hashRes
////				}(n, i, wgInner)
////			}
////			wgInner.Wait()
////			//out <- map[int][]string{n: myRes[n]}
////			fmt.Println(map[int][]string{n: myRes[n]})
////		}(n, wg)
////	}
////	wg.Wait()
////
////}
