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
			return // Прекращаем работу при отмене контекста
		case ch <- num:
			fn(num)
			num++ // Увеличиваем номер
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out) // Закрываем выходной канал при завершении работы
	for num := range in {
		out <- num
		time.Sleep(1 * time.Millisecond) // Имитируем задержку
	}
}

func main() {
	chIn := make(chan int64)

	// Создаем контекст с тайм-аутом.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64   // Сумма сгенерированных чисел
	var inputCount int64 // Количество сгенерированных чисел

	// Запускаем генератор
	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)   // Атомарно обновляем сумму
		atomic.AddInt64(&inputCount, 1) // Атомарно обновляем количество
	})

	const NumOut = 5 // Количество обрабатывающих горутин
	outs := make([]chan int64, NumOut)

	for i := 0; i < NumOut; i++ {
		// Создаем каналы и запускаем worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	var wg sync.WaitGroup
	chOut := make(chan int64, NumOut) // Канал для итоговых результатов

	// Собираем числа из каналов outs
	for i, out := range outs {
		wg.Add(1)
		go func(ch <-chan int64) {
			defer wg.Done()
			for num := range ch {
				chOut <- num
			}
		}(out)
	}

	// Горутина, которая ждет завершения всех worker'ов и закрывает chOut
	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64 // Количество чисел результирующего канала
	var sum int64   // Сумма чисел результирующего канала

	// Читаем числа из результирующего канала
	for num := range chOut {
		atomic.AddInt64(&count, 1)
		atomic.AddInt64(&sum, num)
	}

	// Выводим результаты
	fmt.Println("Количество чисел:", inputCount, "чисел получено:", count)
	fmt.Println("Сумма чисел:", inputSum, "сумма получена:", sum)

	// Проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
}
