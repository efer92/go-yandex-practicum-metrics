# «Сервер сбора метрик и алертинга»

## Инкремент 1

Разработайте сервер для сбора рантайм-метрик, который будет собирать репорты от агентов по протоколу HTTP. Агент вы реализуете в следующем инкременте — в качестве источника метрик вы будете использовать пакет `runtime`.

Сервер должен быть доступен по адресу `http://localhost:8080`, а также:

- Принимать и хранить произвольные метрики двух типов:
  - Тип `gauge`, `float64` — новое значение должно замещать предыдущее.
  - Тип `counter`, `int64` — новое значение должно добавляться к предыдущему, если какое-то значение уже было известно серверу.
- Принимать метрики по протоколу HTTP методом `POST`.
- Принимать данные в формате `http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>`, `Content-Type: text/plain`.
- При успешном приёме возвращать `http.StatusOK`.
- При попытке передать запрос без имени метрики возвращать `http.StatusNotFound`.
- При попытке передать запрос с некорректным типом метрики или значением возвращать `http.StatusBadRequest`.

Редиректы не поддерживаются.

Для хранения метрик объявите тип `MemStorage`. Рекомендуем использовать тип `struct` с полем-коллекцией внутри (`slice` или `map`). В будущем это позволит добавлять к объекту хранилища новые поля — например, логгер или мьютекс, чтобы можно было использовать их в методах. Опишите интерфейс для взаимодействия с этим хранилищем.

Пример запроса к серверу:

```bash
POST /update/counter/someMetric/527 HTTP/1.1
Host: localhost:8080
Content-Length: 0
Content-Type: text/plain 
```

Пример ответа от сервера:

```bash
HTTP/1.1 200 OK
Date: Tue, 21 Feb 2023 02:51:35 GMT
Content-Length: 11
Content-Type: text/plain; charset=utf-8
```

## Инкремент 2

Разработайте агент (HTTP-клиент) для сбора рантайм-метрик и их последующей отправки на сервер по протоколу HTTP.
Агент должен собирать метрики двух типов:

```bash
Тип gauge, float64.
Тип counter, int64.
```

В качестве источника метрик используйте пакет runtime.

Нужно собирать следующие метрики типа gauge:

```bash
Alloc
BuckHashSys
Frees
GCCPUFraction
GCSys
HeapAlloc
HeapIdle
HeapInuse
HeapObjects
HeapReleased
HeapSys
LastGC
Lookups
MCacheInuse
MCacheSys
MSpanInuse
MSpanSys
Mallocs
NextGC
NumForcedGC
NumGC
OtherSys
PauseTotalNs
StackInuse
StackSys
Sys
TotalAlloc
```

К метрикам пакета runtime добавьте ещё две:

```bash
PollCount (тип counter) — счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime (на каждый pollInterval — см. ниже).
RandomValue (тип gauge) — обновляемое произвольное значение.
```

По умолчанию приложение должно:
Обновлять метрики из пакета runtime с заданной частотой: pollInterval — 2 секунды.
Отправлять метрики на сервер с заданной частотой: reportInterval — 10 секунд.

Чтобы приостанавливать работу функции на заданное время, используйте вызов time.Sleep(n * time.Second). Подробнее о пакете time и его возможностях вы узнаете в третьем спринте.

Метрики нужно отправлять по протоколу HTTP методом POST:

```bash
Формат данных — http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>.
Адрес сервера — http://localhost:8080.
Заголовок — Content-Type: text/plain.
```

Пример запроса к серверу:

```bash
POST /update/counter/someMetric/527 HTTP/1.1
Host: localhost:8080
Content-Length: 0
Content-Type: text/plain 
```

Пример ответа от сервера:

```bash
HTTP/1.1 200 OK
Date: Tue, 21 Feb 2023 02:51:35 GMT
Content-Length: 11
Content-Type: text/plain; charset=utf-8 
```

Покройте код агента и сервера юнит-тестами.

## Инкремент 3

Вы написали приложение с помощью пакета стандартной библиотеки net/http. Используя любой внешний пакет (роутер или фреймворк), совместимый с net/http, перепишите ваш код.

Доработайте сервер так, чтобы в ответ на запрос GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ> он возвращал аккумулированное значение метрики в текстовом виде со статусом http.StatusOK.

При попытке запроса неизвестной метрики сервер должен возвращать http.StatusNotFound.

По запросу GET http://<АДРЕС_СЕРВЕРА>/ сервер должен отдавать HTML-страницу со списком имён и значений всех известных ему на текущий момент метрик.

Хендлеры должны взаимодействовать с экземпляром MemStorage при помощи соответствующих интерфейсных методов.

## Начало работы

Выполните:

```bash
git clone https://github.com/efer92/go-yandex-practicum-metrics
git checkout INCREMENT_3
```

Установите зависимости:

```bash
go mod init github.com/efer92/go-yandex-practicum-metrics
go mod tidy
```

Соберите серверную компоненту:

```bash
go build -o server ./cmd/server
```

Запустите сервер:

```bash
./server
```

Запустите агент:

```bash
./agent
```

Откройте в браузере графический интерфейс:

```bash
http://localhost:8080/
```

Запуcтите тесты:

```bash
go test -v ./...
```

Посмотрите покрытие тестами:

```bash
go test -cover ./...
```

Проверки:

```bash
check_status() {
    local url=$1
    local method=$2
    local expected=$3
    local desc=$4
    
    status=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "$url")
    
    if [ "$status" -eq "$expected" ]; then
        echo "✅ $desc: $status (OK)"
    else
        echo "❌ $desc: получили $status, ожидали $expected"
    fi
}

check_status "http://localhost:8080/update/counter/metric/100" "POST" 200 "StatusOK"
check_status "http://localhost:8080/update/counter//100" "POST" 404 "StatusNotFound (пустое имя)"
check_status "http://localhost:8080/update/invalid/test/100" "POST" 400 "StatusBadRequest (неверный тип)"
check_status "http://localhost:8080/update/gauge/test/abc" "POST" 400 "StatusBadRequest (неверное значение)"
```
