package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Интервал очистки кольцевого буфера
const bufferDrainInterval time.Duration = 10 * time.Second

// Размер кольцевого буфера
const bufferSize int = 10

// RingIntBuffer - кольцевой буфер целых чисел
type RingIntBuffer struct {
	array []int
	pos   int
	size  int
	m     sync.Mutex
}

// NewRingIntBuffer - создание нового буфера целых чисел
func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

// Push - добавление нового элемента в буфер,
// если буфер заполнен - самое старое значение затирается
func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
	fmt.Printf("В буфер добавлено число: %d\n", el)
}

// Get - получение всех элементов буфера и его очистка
func (r *RingIntBuffer) Get() []int {
	fmt.Println("Получение всех элементов из буфера и его очистка")
	if r.pos < 0 {
		fmt.Println("Буфер пуст")
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	var output []int = r.array[:r.pos+1]
	r.pos = -1
	fmt.Println("Буфер ощищен")
	return output
}

// StageInt - Стадия конвейера, обрабатывающая целые числа
type StageInt func(<-chan bool, <-chan int) <-chan int

// PipeLineInt - Пайплайн обработки целых чисел
type PipeLineInt struct {
	stages []StageInt
	done   <-chan bool
}

// NewPipelineInt - Создание пайплайна обработки целых чисел
func NewPipelineInt(done <-chan bool, stages ...StageInt) *PipeLineInt {
	return &PipeLineInt{done: done, stages: stages}
}

// Run - Запуск пайплайна для обработки целых чисел
// source - источник данных для конвейера
func (p *PipeLineInt) Run(source <-chan int) <-chan int {
	var c <-chan int = source
	for index := range p.stages {
		c = p.runStageInt(p.stages[index], c)
	}
	fmt.Println("Запущены стадии конвейера")
	return c
}

// runStageInt - запуск стадии конвейера
func (p *PipeLineInt) runStageInt(stage StageInt, sourceChan <-chan int) <-chan int {
	return stage(p.done, sourceChan)
}
func main() {
	// источник данных
	dataSource := func() (<-chan int, <-chan bool) {
		c := make(chan int)
		done := make(chan bool)
		go func() {
			defer close(done)
			scanner := bufio.NewScanner(os.Stdin)
			var data string
			for {
				scanner.Scan()
				data = scanner.Text()
				if strings.EqualFold(data, "exit") {
					fmt.Println("Программа завершила работу!")
					return
				}
				i, err := strconv.Atoi(data)
				if err != nil {
					fmt.Println("Программа обрабатывает только целые числа!")
					continue
				}
				c <- i
			}
		}()
		return c, done
	}
	// стадия, фильтрующая отрицательные числа
	negativeFilterStageInt := func(done <-chan bool, c <-chan int) <-chan int {
		convertedIntChan := make(chan int)
		go func() {
			for {
				select {
				case data := <-c:
					if data > 0 {
						fmt.Printf("Число %d прошло стадию фильтрации отрицательных чисел\n", data)
						convertedIntChan <- data
					} else {
						fmt.Printf("Число %d отфильтровано стадией фильтрации отрицательных чисел\n", data)
					}
				case <-done:
					return
				}
			}
		}()
		return convertedIntChan
	}
	// стадия, фильтрующая числа, не кратные 3
	specialFilterStageInt := func(done <-chan bool, c <-chan int) <-chan int {
		filteredIntChan := make(chan int)
		go func() {
			for {
				select {
				case data := <-c:
					if data > 0 && data%3 == 0 {
						fmt.Printf("Число %d прошло стадию фильтрации чисел кратных 3 и не равных 0\n", data)
						filteredIntChan <- data
					} else {
						fmt.Printf("Число %d отфильтровано стадией фильтрации чисел кратных 3 и не равных 0\n", data)
					}
				case <-done:
					return
				}
			}
		}()
		return filteredIntChan
	}
	// стадия буферизации
	bufferStageInt := func(done <-chan bool, c <-chan int) <-chan int {
		bufferedIntChan := make(chan int)
		buffer := NewRingIntBuffer(bufferSize)
		fmt.Println("Инициализирован буфер целых чисел")
		go func() {
			for {
				select {
				case data := <-c:
					buffer.Push(data)
				case <-done:
					return
				}
			}
		}()
		// стадия, выполняющая получение целых чисел из буфера
		// с заданным интервалом времени - bufferDrainInterval и очищающая буфер
		go func() {
			for {
				select {
				case <-time.After(bufferDrainInterval):
					bufferData := buffer.Get()
					// if bufferData != nil {
					for _, data := range bufferData {
						bufferedIntChan <- data
					}
					// }
				case <-done:
					return
				}
			}
		}()
		return bufferedIntChan
	}
	// Потребитель данных от пайплайна
	consumer := func(done <-chan bool, c <-chan int) {
		for {
			select {
			case data := <-c:
				fmt.Printf("Обработаны данные: %d\n", data)
			case <-done:
				return
			}
		}
	}

	source, done := dataSource()
	pipeline := NewPipelineInt(done, negativeFilterStageInt, specialFilterStageInt, bufferStageInt)
	consumer(done, pipeline.Run(source))
}
