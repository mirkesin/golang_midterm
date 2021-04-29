package main

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var ExecutePipeline = func(jobs ...job) {
	wg := &sync.WaitGroup{}
	var in = make(chan interface{})
	for _, task := range jobs {
		var out = make(chan interface{})
		wg.Add(1)
		go func(wg *sync.WaitGroup, in, out chan interface{}, task job) {
			defer wg.Done()
			defer close(out)
			task(in, out)
		}(wg, in, out, task)
		in = out
	}
	wg.Wait()
}

var SingleHash = func(in, out chan interface{}) {
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		go func(input interface{}, out chan interface{}, wg *sync.WaitGroup, mutex *sync.Mutex) {
			defer wg.Done()

			md5Crc32Chan := make(chan string)
			go func(mutex *sync.Mutex, input interface{}, out chan string) {
				mutex.Lock()
				md5Val := DataSignerMd5(strconv.Itoa(input.(int)))
				mutex.Unlock()
				out <- DataSignerCrc32(md5Val)
			}(mutex, input, md5Crc32Chan)

			crc32Chan := make(chan string)
			go func(input interface{}, out chan string) {
				out <- DataSignerCrc32(strconv.Itoa(input.(int)))
			}(input, crc32Chan)
			crc32Val := <-crc32Chan

			md5Crc32Val := <-md5Crc32Chan
			result := crc32Val + "~" + md5Crc32Val
			out <- result
			runtime.Gosched()
		}(data, out, wg, mutex)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(data string, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			results := make([]string, 6)
			wgItem := &sync.WaitGroup{}
			for index := 0; index <= 5; index++ {
				wgItem.Add(1)
				go func(results []string, index int, data string, wgItem *sync.WaitGroup) {
					defer wgItem.Done()
					data = DataSignerCrc32(strconv.Itoa(index) + data)
					runtime.Gosched()
					results[index] = data
				}(results, index, data, wgItem)
			}
			wgItem.Wait()
			totalResult := strings.Join(results, "")
			out <- totalResult
		}(data.(string), out, wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var result []string
	for data := range in {
		result = append(result, data.(string))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	totalResult := strings.Join(result, "_")
	out <- totalResult
}