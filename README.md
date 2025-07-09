# Scoring API Gateway

GraphQL API сервис для скоринговой системы, который обрабатывает запросы на проверку компаний и взаимодействует с worker-сервисом через NATS.

## Архитектура

- **GraphQL API** - основной интерфейс для фронтенда
- **PostgreSQL** - хранение данных о проверках
- **NATS** - обмен сообщениями с worker-сервисом
- **Zap** - структурированное логгирование
- **Viper** - конфигурация

## Структура проекта

```
scoring_api_gateway/
├── internal/
│   ├── config/     # Конфигурация
│   ├── logger/     # Логгирование
│   ├── repository/ # Доступ к данным
│   ├── messaging/  # NATS клиент
│   ├── service/    # Бизнес-логика
│   └── graphql/    # GraphQL резолверы
├── graph/          # Сгенерированные GraphQL файлы
├── migrations/     # SQL миграции
├── config.yaml     # Конфигурация
└── main.go         # Точка входа
```

## Установка и запуск

### Предварительные требования

- Go 1.21+
- PostgreSQL
- NATS Server

### Настройка базы данных

1. Создайте базу данных:

```sql
CREATE DATABASE scoring;
```

2. Примените миграции:

```sql
\c scoring
\i migrations/001_init.sql
```

### Настройка NATS

Запустите NATS сервер:

```bash
nats-server
```

### Запуск приложения

1. Установите зависимости:

```bash
go mod tidy
```

2. Запустите приложение:

```bash
go run main.go
```

Сервер будет доступен по адресу: http://localhost:8080

## GraphQL API

### Создание проверки

```graphql
mutation CreateVerification(
  $inn: String!
  $requestedDataTypes: [VerificationDataType!]!
) {
  createVerification(inn: $inn, requestedDataTypes: $requestedDataTypes) {
    id
    inn
    status
    authorEmail
    requestedDataTypes
    createdAt
    updatedAt
  }
}
```

Пример запроса:

```json
{
  "inn": "7707083893",
  "requestedDataTypes": ["BASIC_INFO", "FINANCIAL_METRICS"]
}
```

### Получение статуса проверки

```graphql
query GetVerification($id: ID!) {
  verification(id: $id) {
    id
    inn
    status
    authorEmail
    requestedDataTypes
    data {
      dataType
      data
      createdAt
    }
    createdAt
    updatedAt
  }
}
```

## Конфигурация

Настройки можно изменить в файле `config.yaml` или через переменные окружения:

- `SERVER_HOST` - хост сервера
- `SERVER_PORT` - порт сервера
- `DATABASE_HOST` - хост PostgreSQL
- `DATABASE_PORT` - порт PostgreSQL
- `DATABASE_USER` - пользователь PostgreSQL
- `DATABASE_PASSWORD` - пароль PostgreSQL
- `DATABASE_DBNAME` - имя базы данных
- `NATS_URL` - URL NATS сервера
- `LOG_LEVEL` - уровень логгирования
- `LOG_JSON` - формат логов (JSON/текст)

## Разработка

### Генерация GraphQL кода

После изменения схемы GraphQL:

```bash
go run github.com/99designs/gqlgen generate
```

### Тестирование

```bash
go test ./...
```

## Логгирование

Приложение использует структурированное логгирование с помощью Zap. Логи включают:

- Информацию о запросах
- Ошибки подключения к БД/NATS
- Статус операций с проверками
