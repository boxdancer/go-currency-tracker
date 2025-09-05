# go-currency-tracker

## 🛠 Запуск контейнеров
```bash
docker-compose up -d
```

## 🛠 Запуск линтера
```bash
golangci-lint run
```
## 🛠 Прогнать тесты в проекте
```bash
go test -race ./...
```

Проект на Go для отслеживания курсов валют и криптовалют.

## 🚀 Стек технологий (план)
- **Go** — основной язык
- **Redis** — кэширование
- **Docker / Docker Compose** — контейнеризация
- **PostgreSQL** — хранение пользователей (на следующих этапах)
- **Kafka** — обмен сообщениями между сервисами (на следующих этапах)
- **gRPC** — внутреннее общение между сервисами (на следующих этапах)
- **Prometheus + Grafana** — метрики (на следующих этапах)

---

## 📌 Этап 1 — MVP монолит

На данном этапе проект представляет собой сервис, который:
1. Запрашивает курс валюты/криптовалюты через внешний API (CoinGecko).
2. Отдаёт результат через REST API.
3. Имеет эндпоинт для проверки работы.

---

## 🔗 REST API

### Prometheus GUI
`http://localhost:9090/query`

### Profiling
```http
GET /debug/pprof/
```
`http://localhost:8080/debug/pprof/`


### Метрики Prometheus
```http
GET /metrics
```
`http://localhost:8080/metrics`

**Пример ответа:**
```
...
cached_client_cache_hits_total 50
cached_client_cache_misses_total 3
...
```

### Проверка сервера
```http
GET /ping
```
`http://localhost:8080/ping`

**Пример ответа:**
```
pong
```

### Получение курса BTC/USD
```http
GET /btc-usd
```
`http://localhost:8080/btc-usd`

**Пример ответа:**
```
BTC/USD: 29341.00
```

### Конкурентное получение нескольких курсов
```http
GET /rates
```
`http://localhost:8080/rates`

**Пример ответа:**
```
{"bitcoin":{"usd":109658},"ethereum":{"usd":4306.79},"usd":{"rub":81.31}}
```

---

## 🛠 Запуск локально

```bash
go run cmd/app/main.go
```

Сервер стартует на `http://localhost:8080`.

---

## 📅 Планы по развитию
- Добавить конкурентное получение курсов нескольких валют.
- Добавить кэширование в Redis.
- Реализовать Profile Service с авторизацией.
- Подключить Kafka для событий.
- Добавить gRPC для внутреннего общения сервисов.
- Настроить метрики (Prometheus + Grafana).