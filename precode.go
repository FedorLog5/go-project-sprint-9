package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)


func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch)
	var num int64 = 1
	for {
		select {
		case <-ctx.Done(): 
			return
		default:
			ch <- num
			fn(num) 
			num++
		}
	}
}


func Worker(in <-chan int64, out chan<- int64) {
	defer close(out)
	for v := range in { 
		out <- v
		time.Sleep(1 * time.Millisecond) 
	}
}

func main() {
	chIn := make(chan int64)

	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64   
	var inputCount int64 
	
	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i) 
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5 
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup


	for i := 0; i < NumOut; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for v := range outs[i] {
				chOut <- v
				atomic.AddInt64(&amounts[i], 1)
			}
		}(i)
	}

	go func() {
		wg.Wait() 
		close(chOut)
	}()

	var count int64 
	var sum int64   

	
	for v := range chOut {
		atomic.AddInt64(&sum, v)
		atomic.AddInt64(&count, 1)
	}

	fmt.Println("Количество чисел", atomic.LoadInt64(&inputCount), atomic.LoadInt64(&count))
	fmt.Println("Сумма чисел", atomic.LoadInt64(&inputSum), atomic.LoadInt64(&sum))
	fmt.Println("Разбивка по каналам", amounts)


	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
