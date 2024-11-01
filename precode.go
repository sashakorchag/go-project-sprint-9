package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1, 2, 3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch) // Закрываем канал при выходе из функции.

	var num int64 = 1
	for {
		select {
		case <-ctx.Done():
			return
		case ch <- num:
			fn(num)
			num++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out) // Закрываем результирующий канал при выходе из функции.

	for num := range in { // Используем цикл по каналу.
		out <- num
		time.Sleep(1 * time.Millisecond) // Имитируем задержку.
	}
}

func main() {
	chIn := make(chan int64)

	// Создаем контекст с тайм-аутом.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64   // Сумма сгенерированных чисел.
	var inputCount int64 // Количество сгенерированных чисел.

	// Запускаем генератор, который будет подсчитывать сумму и количество.
	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)   // Используем атомарный доступ для суммы.
		atomic.AddInt64(&inputCount, 1) // Используем атомарный доступ для количества.
	})

	const NumOut = 5 // Количество обрабатывающих горутин.
	outs := make([]chan int64, NumOut)

	for i := 0; i < NumOut; i++ {
		// Создаем каналы и запускаем worker.
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	chOut := make(chan int64, NumOut) // Канал для итоговых результатов.
	var wg sync.WaitGroup

	// Собираем числа из каналов outs.
	for _, out := range outs {
		wg.Add(1)
		go func(ch <-chan int64) {
			defer wg.Done()
			for num := range ch {
				chOut <- num
			}
		}(out)
	}

	go func() {
		wg.Wait()    // Ждем завершения всех worker'ов.
		close(chOut) // Закрываем результирующий канал.
	}()

	var count int64 // Количество чисел результирующего канала.
	var sum int64   // Сумма чисел результирующего канала.

	// Читаем числа из результирующего канала.
	for num := range chOut {
		atomic.AddInt64(&count, 1) // Используем атомарный доступ для количества.
		atomic.AddInt64(&sum, num) // Используем атомарный доступ для суммы.
	}

	// Выводим результаты
	fmt.Println("Количество чисел:", inputCount, "количество из chOut:", count)
	fmt.Println("Сумма чисел:", inputSum, "сумма из chOut:", sum)

	// Проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
}
