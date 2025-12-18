## Профилирование приложений

### Go имеет мощный встроенный профайлер, который поддерживает профилирование CPU, памяти, горутин и блокировок.

<hr style="border: 1px solid orange;"/>

### Подключаение профайлера

Всё, что вам нужно для подключения профайлера, – импортировать net/http/pprof; необходимые HTTP-обработчики будут зарегистрированы автоматически:

В качестве примера используется сервис-консьюмер [city-ranker-service](https://gitlab.almanit.kz/jmart/city-ranker-service). 
Данный сервис получает сообщения из кролика, на своей стороне валидирует и записывает в базу данных.

```go
func main() {
	go func() {
		r := http.NewServeMux()

		// Регистрация pprof-обработчиков
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)

		http.ListenAndServe(":8080", r)
	}()

	app, err := app.NewApp()

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		app.RunConsole()
	}()

	if err := app.RunConsume(); err != nil {
		log.Fatal(err)
	}
}

```
<hr style="border: 1px solid orange;"/>

### Профилирование CPU
Во время нагрузки запускаем данную команду
```bash
 go tool pprof  goprofex http://127.0.0.1:8080/debug/pprof/profile
```
Профайлер CPU по умолчанию работает в течение 30 секунд. Он использует выборку, чтобы определить, какие функции тратят большую часть процессорного времени. Рантайм Go останавливает выполнение каждые десять миллисекунд и записывает текущий стек вызовов всех работающих горутин.


Далее открывается интерактивый режим, где можно ввести <code>top</code>

```bash
(pprof) top
Showing nodes accounting for 970ms, 43.89% of 2210ms total
Dropped 89 nodes (cum <= 11.05ms)
Showing top 10 nodes out of 174
      flat  flat%   sum%        cum   cum%
     390ms 17.65% 17.65%      390ms 17.65%  internal/runtime/syscall.Syscall6
     200ms  9.05% 26.70%      200ms  9.05%  runtime.futex
      90ms  4.07% 30.77%      280ms 12.67%  runtime.mallocgc
      60ms  2.71% 33.48%       60ms  2.71%  runtime.nextFreeFast (inline)
      60ms  2.71% 36.20%      150ms  6.79%  runtime.selectgo
      40ms  1.81% 38.01%       40ms  1.81%  runtime.(*mspan).writeHeapBitsSmall
      40ms  1.81% 39.82%       40ms  1.81%  runtime.memmove
      30ms  1.36% 41.18%       30ms  1.36%  indexbytebody
      30ms  1.36% 42.53%       50ms  2.26%  log.formatHeader
      30ms  1.36% 43.89%       30ms  1.36%  runtime.findfunc
```

Команда <code>top</code> выводит писок функций, которые в процентном соотношнии больше всего присутствовали в полученной выборке.
Как мы видим это стандартные библиотеки. Для нас это не информативно.

С помощью команды <code>list</code> можно подробно исследовать каждую функцию. Например:

```bash
(pprof) list ConsumeOffers     
Total: 2.21s
ROUTINE ======================== gitlab.almanit.kz/jmart/city-ranker-service/internal/modules/handler/mq.(*OfferHandler).ConsumeOffers in /home/aabdranbayev/Рабочий стол/workers/city-ranker-service/internal/modules/handler/mq/init.go
         0       80ms (flat, cum)  3.62% of Total
         .          .     19:func (o *OfferHandler) ConsumeOffers(ctx context.Context, msg *message.Message) error {
         .          .     20:   var body structures.MqOffersRequest
         .          .     21:
         .       70ms     22:   if err := json.Unmarshal([]byte(string(msg.Payload)), &body); err != nil {
         .          .     23:           return err
         .          .     24:   }
         .          .     25:
         .       10ms     26:   if err := o.productService.Insert(ctx, body); err != nil {
         .          .     27:           return err
         .          .     28:   }
         .          .     29:
         .          .     30:   return nil
         .          .     31:}
(pprof) 
```
Расшифровка:
* ROUTINE ======================== gitlab.almanit.kz/jmart/city-ranker-service/internal/modules/handler/mq.(*OfferHandler).ConsumeOffers in /home/aabdranbayev/Рабочий стол/workers/city-ranker-service/internal/modules/handler/mq/init.go <br>
Имя функции которую профайлер анализировал.
* Total: 2.21s.  <br>Всего было зафиксировано 2.21 секунды процессорного времени в этом профилировании
*   0       80ms (flat, cum)  3.62% of Total  <br>
    Вся функция (включая вызовы других функций внутри) использовала 80ms процессорного времени.
*  70ms     22:   if err := json.Unmarshal([]byte(string(msg.Payload)), &body); err != nil  <br>
  Тут мы видим что основное время 70ms из 80ms тратится на разбор JSON
* 10ms     26:   if err := o.productService.Insert(ctx, body); err != nil {  <br>
  Тут мы видим что 10ms из 80ms тратится на вставку (Время настолько маленькое так как внутри этой функии есть проверка на длину, и мы не дошли до базы)

Команда <code>web</code> генерирует граф вызовов в формате SVG и открывает его в веб-браузере.

<hr style="border: 1px solid orange;"/>

### Профилирование Кучи

Во время нагрузки запускаем данную команду
```bash
 go tool pprof  goprofex http://127.0.0.1:8080/debug/pprof/heap
```

По умолчанию он показывает объём используемой памяти:

```bash
(pprof) top
Showing nodes accounting for 2210.07kB, 100% of 2210.07kB total
Showing top 10 nodes out of 21
      flat  flat%   sum%        cum   cum%
  641.34kB 29.02% 29.02%   641.34kB 29.02%  github.com/microsoft/go-mssqldb/internal/cp.init
  544.67kB 24.64% 53.66%   544.67kB 24.64%  regexp/syntax.(*compiler).inst (inline)
  512.05kB 23.17% 76.83%   512.05kB 23.17%  github.com/gabriel-vasile/mimetype.newMIME (inline)
  512.02kB 23.17%   100%   512.02kB 23.17%  regexp/syntax.(*parser).maybeConcat
         0     0%   100%   512.05kB 23.17%  github.com/gabriel-vasile/mimetype.init
         0     0%   100%   512.02kB 23.17%  github.com/jinzhu/inflection.compile
         0     0%   100%   512.02kB 23.17%  github.com/jinzhu/inflection.init.0
         0     0%   100%   544.67kB 24.64%  go.opencensus.io/trace/tracestate.init
         0     0%   100%  1056.68kB 47.81%  regexp.Compile (inline)
         0     0%   100%  1056.68kB 47.81%  regexp.MustCompile
(pprof) 
```

<code>-alloc_objects</code>

Если интересует кол-во размещенных в куче объектов. Запускаем с опцией <code>-alloc_objects</code>.Этот профиль отображает количество объектов, выделенных в памяти (не их размер, а именно количество аллокаций). Полезно для выявления горячих точек, где происходит много операций выделения памяти.

```bash
 go tool pprof -alloc_objects goprofex http://127.0.0.1:8080/debug/pprof/heap
Fetching profile over HTTP from http://127.0.0.1:8080/debug/pprof/heap
goprofex: stat goprofex: no such file or directory
Fetched 1 source profiles out of 2
Saved profile in /home/aabdranbayev/pprof/pprof.___go_build_gitlab_almanit_kz_jmart_city_ranker_service_cmd_app.alloc_objects.alloc_space.inuse_objects.inuse_space.005.pb.gz
File: ___go_build_gitlab_almanit_kz_jmart_city_ranker_service_cmd_app
Type: alloc_objects
Time: Nov 22, 2024 at 12:47pm (+05)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 1863436, 71.30% of 2613583 total
Dropped 44 nodes (cum <= 13067)
Showing top 10 nodes out of 72
      flat  flat%   sum%        cum   cum%
    781485 29.90% 29.90%    1171533 44.82%  github.com/ThreeDotsLabs/watermill.(*StdLoggerAdapter).log
    291723 11.16% 41.06%     291723 11.16%  fmt.Sprintf
    131073  5.02% 46.08%     131073  5.02%  encoding/binary.Read
    127181  4.87% 50.94%     127181  4.87%  github.com/ThreeDotsLabs/watermill/message.NewMessage
    116514  4.46% 55.40%     214819  8.22%  github.com/rabbitmq/amqp091-go.readShortstr
     98325  3.76% 59.16%      98325  3.76%  log.(*Logger).output
     92478  3.54% 62.70%     136174  5.21%  gitlab.almanit.kz/jmart/city-ranker-service/internal/modules/handler/mq.(*OfferHandler).ConsumeOffers
     89212  3.41% 66.12%     371024 14.20%  github.com/rabbitmq/amqp091-go.readTable
     69908  2.67% 68.79%      69908  2.67%  github.com/rabbitmq/amqp091-go.readLongstr
     65537  2.51% 71.30%      91753  3.51%  context.WithCancel
(pprof) 

```
Расшифровка:
* Total: 2,613,583. Общее количество объектов, выделенных в приложении во время профилирования.
* Flat и Cumulative (cum):
  * Flat: Количество аллокаций, напрямую сделанных в конкретной функции (без учета вызовов других функций из нее).
  * Cumulative (cum): Общее количество аллокаций, включая вызовы других функций из данной.
* Как мы видим <code>github.com/ThreeDotsLabs/watermill.(*StdLoggerAdapter).log</code>. Эта функция явно выделяет много объектов (почти половина всех аллокаций). Вероятно, вы используете подробное логирование, что вызывает нагрузки.


<code>-inuse_objects</code>

<code>-alloc_space</code>

```bash
 go tool pprof -alloc_space goprofex http://127.0.0.1:8080/debug/pprof/heap
(pprof) top
Showing nodes accounting for 61.01MB, 72.17% of 84.54MB total
Showing top 10 nodes out of 152
      flat  flat%   sum%        cum   cum%
      18MB 21.29% 21.29%       34MB 40.22%  github.com/ThreeDotsLabs/watermill.(*StdLoggerAdapter).log
      10MB 11.83% 33.12%       10MB 11.83%  fmt.Sprintf
       6MB  7.10% 40.22%        8MB  9.47%  gitlab.almanit.kz/jmart/city-ranker-service/internal/modules/handler/mq.(*OfferHandler).ConsumeOffers
       6MB  7.10% 47.32%        6MB  7.10%  log.(*Logger).output
       5MB  5.92% 53.24%     6.50MB  7.69%  github.com/rabbitmq/amqp091-go.(*Channel).recvContent
    3.50MB  4.14% 57.38%        6MB  7.10%  github.com/rabbitmq/amqp091-go.(*reader).parseMethodFrame
    3.50MB  4.14% 61.52%     7.50MB  8.87%  github.com/rabbitmq/amqp091-go.readTable
    3.50MB  4.14% 65.66%     3.50MB  4.14%  github.com/rabbitmq/amqp091-go.readShortstr
       3MB  3.55% 69.21%        3MB  3.55%  github.com/ThreeDotsLabs/watermill.LogFields.Add
    2.50MB  2.96% 72.17%     2.50MB  2.96%  github.com/rabbitmq/amqp091-go.(*reader).parseBodyFrame
(pprof) 
```
Расшифровка:
* Type: alloc_space
  Этот профиль показывает объем памяти (в байтах), выделенной различными функциями во время выполнения приложения. Это полезно для понимания, где больше всего используется память.
* Total: 84.54 MB
  Общий объем памяти, выделенной в приложении во время профилирования.
* Flat и Cumulative (cum):
    *  Flat: Количество памяти, выделенной напрямую в конкретной функции.
    * Cumulative (cum): Суммарное количество памяти, включая вызовы других функций из этой.
* github.com/ThreeDotsLabs/watermill.(StdLoggerAdapter).log
  Логирование. Эта функция вызывает значительные аллокации памяти (40% от общего объема). Возможно, связано с подробным логированием.

<hr style="border: 1px solid orange;"/>

### Профилирование горутин


<hr style="border: 1px solid orange;"/>


### Профилирование блокировок

<hr style="border: 1px solid orange;"/>

### Бенчмарки