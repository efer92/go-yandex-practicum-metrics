# «Сервер сбора метрик и алертинга»

## Задание. Инкремент 1

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

## Начало работы

Выполните:

```
git clone https://github.com/efer92/go-yandex-practicum-metrics
git checkout INCREMENT_1
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
go build -o server ./cmd/server
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
