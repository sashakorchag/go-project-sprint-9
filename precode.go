package main
import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	// ...
	var num int64 = 1
	for true {
		select {
		case ch <- num:
			fn(num)
			atomic.AddInt64(&num, 1)
		case <-ctx.Done():
			close(ch)
			return
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	for true {
		num, ok := <-in
		if !ok {
			close(out)
			return
		}
		out <- num
		time.Sleep(1 * time.Millisecond)
	}
}

func main() {
	chIn := make(chan int64)

	var wgCtx sync.WaitGroup
	wgCtx.Add(1)
	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	// Предотвращаем программу от завершения до завершения работы горутин
	context.AfterFunc(ctx, func() {
		wgCtx.Done()
		return
	})
	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum += i
		inputCount++
		atomic.AddInt64(&inputSum, i)
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}
	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	wg.Add(1)
	// 4. Собираем числа из каналов outs
	for i, out := range outs {
		go func(ch <-chan int64, i int) {
			wg.Add(1)
			defer wg.Done()
			for true {
				num, ok := <-ch
				if !ok {
					return
				}
				atomic.AddInt64(&amounts[i], 1)
				chOut <- num
			}

		}(out, i)
	}
	wg.Done()

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()
	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	go func(ch <-chan int64) {
		wgCtx.Add(1)
		defer wgCtx.Done()
		for true {
			num, ok := <-ch
			if !ok {
				return
			}
			atomic.AddInt64(&count, 1)
			atomic.AddInt64(&sum, num)
		}

	}(chOut)

	wgCtx.Wait()

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	fmt.Println(inputSum, sum)
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}