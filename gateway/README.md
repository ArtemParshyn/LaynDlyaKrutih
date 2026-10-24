# Gateway Service

Шлюз входа системы прогноза денежных потоков на Go (Gin).
Отвечает за авторизацию (JWT + Redis + Argon2id), извлечение `user_id` из токенов и оркестрацию микросервисов:
- **Выписка (Ledger)** (`LEDGER_SERVICE_URL`)
- **Планы (Planning)** (`PLANNING_SERVICE_URL`)
- **Прогноз (Forecast)** (`FORECAST_SERVICE_URL`)

Вход **не хранит** операции и **не считает** дни самостоятельно.

---

## Архитектура Gateway
```
gateway/
├── auth/                 # JWT токены, пароли Argon2id, AuthMiddleware
├── client/
│   ├── ledger.go         # Клиент к микросервису выписки
│   ├── planning.go       # Клиент к микросервису планов
│   └── forecast.go       # Клиент к микросервису прогноза
├── database/             # PostgreSQL (только таблица users) и Redis
├── dto/
│   ├── user.go           # DTO авторизации и профиля
│   └── forecast.go       # Финансовый пакет, дни, алерты, сценарии
├── handlers/
│   ├── auth_handler.go   # Регистрация, логин, refresh, logout
│   ├── user_handler.go   # CRUD пользователей и профиль
│   └── forecast_handler.go # Statements, Forecast, Events, Scenarios
├── migrations/           # Миграции таблицы users
├── repository/           # Репозиторий пользователей
├── service/
│   └── orchestrator.go   # Логика сборки пакета и оркестрации сервисов
├── Dockerfile
└── main.go
```

---

## Эндпоинты

### 1. Авторизация (Публичные):
| Метод | URL | Описание |
|---|---|---|
| `POST` | `/register` | Регистрация (email, password, confirm_password, company_name) |
| `POST` | `/login` | Вход, выдача токенов с `company_name` внутри JWT |
| `POST` | `/refresh` | Продление сессии по refresh токену |
| `POST` | `/logout` | Завершение сессии |

### 2. Оркестрация Прогноза (Требуется `Authorization: Bearer <token>`):
| Метод | URL | Описание |
|---|---|---|
| `POST` | `/api/statements` | Загружает CSV выписке, архивирует старую линию, собирает пакет (выписка + планы) → считает прогноз → сохраняет в выписку |
| `GET`  | `/api/forecast` | Читает уже посчитанные 30 дней и предупреждения из выписки. Не пересчитывает |
| `POST` | `/api/events` | Сохраняет событие в планах и запускает полный пересчёт прогноза |
| `POST` | `/api/scenarios` | Считает альтернативный сценарий «что если» (вторая линия). В выписке не сохраняет |

---

## Переменные окружения
```env
PORT=8080
JWT_SECRET=super_secret_jwt_key
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=secret123
DB_NAME=auth_db
REDIS_HOST=localhost
REDIS_PORT=6379

# Микросервисы:
LEDGER_SERVICE_URL=http://ledger:8081
PLANNING_SERVICE_URL=http://planning:8082
FORECAST_SERVICE_URL=http://forecast:8000
```

## 1C integration

The gateway provides a separate machine-to-machine REST API. It uses the same
orchestrator as the web application, so statement parsing, recurring-payment
detection and forecast calculation remain identical.

Set `ONEC_INTEGRATION_TOKEN` in the gateway environment. Send that value in
`X-1C-Token` (or `Authorization: Bearer ...`) and send the target application
user in `X-User-ID`.

Endpoints (the `/api/integration/1c` prefix is an equivalent alias):

* `GET /integration/1c/health`
* `POST /integration/1c/statements` — multipart field `file`, or raw CSV body;
  CP1251/UTF-8/UTF-16, semicolon/comma/tab exports are supported by ledger.
* `GET /integration/1c/forecast` — current forecast, operations and history.
* `GET /integration/1c/events` — planned events.
* `POST /integration/1c/events` — JSON event in the same format as the web API.

Example headers:

```text
X-1C-Token: <ONEC_INTEGRATION_TOKEN>
X-User-ID: 42
```
