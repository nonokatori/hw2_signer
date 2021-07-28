package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go test -v -race ./...

func ExecutePipeline(jobs4run ...job) {
	in := make(chan interface{})
	var wg sync.WaitGroup
	for _, j := range jobs4run {
		out := make(chan interface{})
		wg.Add(1)
		go executor(j, in, out, &wg)
		in = out
	}
	wg.Wait()
}

func executor(j job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	start := time.Now()
	j(in, out)
	end := time.Since(start)
	fmt.Println(end)
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	var mu = &sync.Mutex{}
	for inputValue := range in {
		ch := make(chan interface{}, 1)
		wg.Add(1)
		go shWorker(inputValue, ch, mu, &wg)
		out = ch
	}
	wg.Wait()
}

func shWorker(inputValue interface{}, ch chan interface{}, mu *sync.Mutex, group *sync.WaitGroup) {
	defer group.Done()
	//defer close(ch)
	var value string
	ch1 := make(chan interface{}, 1)
	ch2 := make(chan interface{}, 1)
	switch inputValue.(type) {
	case int:
		value = strconv.Itoa(inputValue.(int))
	case string:
		value = inputValue.(string)
	}
	mu.Lock()
	res2 := DataSignerMd5(value)
	mu.Unlock()
	go crc32work(ch1, value)
	go crc32work(ch2, res2)
	res := (<-ch1).(string) + "~" + (<-ch2).(string)
	fmt.Println(res)
	close(ch1)
	close(ch2)
	ch <- res
}

func crc32work(ch chan interface{}, value string) {
	ch <- DataSignerCrc32(value)
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	var value string
	fmt.Println("im here")
	for inputValue := range in {
		switch inputValue.(type) {
		case int:
			value = strconv.Itoa(inputValue.(int))
		case string:
			value = inputValue.(string)
		}
		fmt.Println(value)
		ch := make(chan interface{}, 1)
		wg.Add(1)
		go mhWorker(value, ch, &wg)
		out <- ch
	}
	wg.Wait()
}

func mhWorker(value string, out chan interface{}, group *sync.WaitGroup) {
	defer group.Done()
	defer close(out)
	wg := &sync.WaitGroup{}
	var res string
	for i := 0; i < 5; i++ {
		ch := make(chan interface{}, 1)
		wg.Add(1)
		go multiWork(i, value, ch, wg)
		res += (<-ch).(string)
		fmt.Println(<-ch)
	}
	wg.Wait()
	fmt.Println(res)
	out <- 0
}

func resultAdd(in, out chan interface{}) {

}

func multiWork(i int, val string, ch chan interface{}, group *sync.WaitGroup) {
	defer group.Done()
	ch <- DataSignerCrc32(strconv.Itoa(i) + val)
}

func CombineResults(in, out chan interface{}) {
	var results []string
	for res := range in {
		r, ok := res.(string)
		if !ok {
			continue
		}
		results = append(results, r)
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}

func main() {

	inputData := []int{0, 1}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		//job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				println("cant convert result data to string")
			}
			testResult := data
			println(testResult)
		}),
	}
	ExecutePipeline(hashSignJobs...)
}
