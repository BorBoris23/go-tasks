package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// вот тут я немного поплы что внутри функций обьявлять что вне - я так понял обьявление внефункций это что то наподобии глобальной переменно для пакета
var md5Mutex sync.Mutex

// напилил структуру специально для тогочто бы попилить структуры  и поюзать их
type hashResult struct {
	crc32Data string
	crc32Md5  string
}

func ExecutePipeline(jobs ...job) {

	in, out := make(chan interface{}), make(chan interface{})

	var wg sync.WaitGroup

	runJobs(jobs, in, out, &wg)

	wg.Wait()
}

func runJobs(
	jobs []job,
	in, out chan interface{},
	wg *sync.WaitGroup,
) {
	// с вейт группой тоже бы поговорил
	wg.Add(len(jobs))

	for _, worker := range jobs {
		// не совсе понял пролбему в передаче аргументов а вернее в том что есть проблема когда их не передаем
		//go func() {
		// 	worker(in, out)
		// }()
		go func(in, out chan interface{}, worker job) {
			defer wg.Done()

			worker(in, out)
			close(out)
		}(in, out, worker)

		in = out
		out = make(chan interface{})
	}
}

func SingleHash(in, out chan interface{}) {

	var wg sync.WaitGroup

	for data := range in {

		wg.Add(1)

		go func(data interface{}) {
			defer wg.Done()

			out <- calculateSingleHash(data)

		}(data)
	}

	wg.Wait()
}

func calculateSingleHash(data interface{}) string {

	num := data.(int)

	dataStr := strconv.Itoa(num)

	var crc32Data string
	var crc32Md5 string

	var innerWg sync.WaitGroup

	innerWg.Add(2)

	go func() {
		defer innerWg.Done()

		crc32Data = DataSignerCrc32(dataStr)
	}()

	go func() {
		defer innerWg.Done()

		md5Mutex.Lock()

		md5Data := DataSignerMd5(dataStr)

		md5Mutex.Unlock()

		crc32Md5 = DataSignerCrc32(md5Data)
	}()

	innerWg.Wait()

	return crc32Data + "~" + crc32Md5
}

// вот тут помогла иишка саму идею - ибо я пробовал через вайт группу делать
// реализация сбокри структыр для получения обоих результатов
// func SingleHash(in, out chan interface{}) {

// 	resultCh := make(chan hashResult)

// 	for data := range in {

// 		num := data.(int)
// 		dataStr := strconv.Itoa(num)

// 		go calculateCrc32(dataStr, resultCh)

// 		go calculateMd5Crc32(dataStr, resultCh)

// 		var crc32Data string
// 		var crc32Md5 string

// 		for i := 0; i < 2; i++ {
// 			result := <-resultCh

// 			if result.crc32Data != "" {
// 				crc32Data = result.crc32Data
// 			}

// 			if result.crc32Md5 != "" {
// 				crc32Md5 = result.crc32Md5
// 			}
// 		}

// 		out <- crc32Data + "~" + crc32Md5
// 	}
// }

//вариант через вайт группу
// func SingleHash(in, out chan interface{}) {

// 	for data := range in {

// 		num := data.(int)
// 		dataStr := strconv.Itoa(num)

// 		var crc32Data string
// 		var crc32Md5 string

// 		var wg sync.WaitGroup
// 		wg.Add(2)

// 		go func() {
// 			defer wg.Done()

// 			crc32Data = DataSignerCrc32(dataStr)
// 		}()

// 		go func() {
// 			defer wg.Done()

// 			md5Data := DataSignerMd5(dataStr)
// 			crc32Md5 = DataSignerCrc32(md5Data)
// 		}()

// 		wg.Wait()

// 		out <- crc32Data + "~" + crc32Md5
// 	}
// }

func calculateCrc32(dataStr string, resultCh chan hashResult) {
	crc32Data := DataSignerCrc32(dataStr)

	resultCh <- hashResult{
		crc32Data: crc32Data,
	}
}

func calculateMd5Crc32(dataStr string, resultCh chan hashResult) {
	md5Data := DataSignerMd5(dataStr)

	resultCh <- hashResult{
		crc32Md5: DataSignerCrc32(md5Data),
	}
}

func MultiHash(in, out chan interface{}) {

	var wg sync.WaitGroup

	for data := range in {

		wg.Add(1)

		go func(data interface{}) {
			defer wg.Done()

			dataStr := data.(string)

			result := calculateMultiHash(dataStr)

			out <- result

		}(data)
	}

	wg.Wait()
}

func calculateMultiHash(data string) string {
	hashes := make([]string, 6)

	var wg sync.WaitGroup
	wg.Add(6)

	for i := 0; i < 6; i++ {
		go func(i int) {
			defer wg.Done()

			hashes[i] = DataSignerCrc32(strconv.Itoa(i) + data)
		}(i)
	}

	wg.Wait()

	return strings.Join(hashes, "")
}

func CombineResults(in, out chan interface{}) {
	results := []string{}

	for data := range in {
		results = append(results, data.(string))
	}

	sort.Strings(results)

	out <- strings.Join(results, "_")
}
