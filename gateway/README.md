# Auth & User Service

Микросервис авторизации и CRUD пользователей на Go (Gin, GORM, PostgreSQL, Redis, Argon2id, JWT). Без сторонних зависимостей вроде брокеров очередей.

## Стек технологий
- **Go 1.26** / **Gin Framework**
- **PostgreSQL 16** + **GORM** (хранение пользователей)
- **Redis 7** (кэширование пользователей и хранение сессий refresh токенов)
- **Argon2id** (криптографическое хеширование паролей)
- **JWT (golang-jwt/v5)** (Access токены с валидацией через Middleware)
- **Golang Migrate** (миграции базы данных)
- **Docker & Docker Compose**

---

## Архитектура проекта
```
gateway/
├── auth/
│   ├── jwt.go            # Выпуск, валидация JWT, генерация refresh-токена и AuthMiddleware
│   └── password.go       # Хеширование и верификация паролей через Argon2id
├── database/
│   ├── postgres.go       # Подключение к PostgreSQL через GORM
│   └── redis.go          # Подключение к Redis
├── dto/
│   └── user.go           # Request / Response структуры с валидацией (binding)
├── entity/
│   └── user.go           # GORM-модель пользователя (id, email, company_name, password...)
├── handlers/
│   ├── auth_handler.go   # Регистрация, логин, refresh, logout
│   └── user_handler.go   # GetMe, GetUser, GetAll, UpdateUser, DeleteUser + кэширование
├── migrations/
│   ├── 000001_create_users_table.up.sql   # Создание таблицы users сразу с company_name
│   └── 000001_create_users_table.down.sql
├── repository/
│   └── user_repo.go      # Слой работы с БД через GORM интерфейс
├── .env                  # Локальные переменные окружения
├── docker-compose.yml    # Запуск PostgreSQL, Redis, Migration и самого приложения
├── Dockerfile            # Multi-stage сборка Go приложения
└── main.go               # Инициализация роутов и старт сервера
```

---

## API Эндпоинты

### Публичные (без токена):
| Метод | URL | Описание |
|---|---|---|
| `POST` | `/register` | Регистрация нового пользователя |
| `POST` | `/login` | Вход, выдача Access (15 мин) и Refresh (7 дней) токенов + куки |
| `POST` | `/refresh` | Выпуск нового Access токена по Refresh токену (из куки или body) |
| `POST` | `/logout` | Удаление сессии из Redis и очистка кук |
| `GET`  | `/health` | Проверка здоровья сервиса |

### Защищенные (требуется `Authorization: Bearer <token>` или кука `access_token`):
| Метод | URL | Описание |
|---|---|---|
| `GET`    | `/me` | Профиль текущего авторизованного пользователя |
| `GET`    | `/users` | Список всех пользователей (с кэшированием в Redis) |
| `GET`    | `/users/:id` | Пользователь по ID (с кэшированием в Redis) |
| `PATCH`  | `/users/:id` | Обновление `company_name` и/или `password` (сброс кэша) |
| `DELETE` | `/users/:id` | Удаление пользователя (сброс кэша) |

---

## Примеры запросов

### 1. Регистрация (`POST /register`)
```json
{
  "email": "developer@company.com",
  "password": "strongPassword123",
  "company_name": "Acme Corp"
}
```

### 2. Вход (`POST /login`)
```json
{
  "email": "developer@company.com",
  "password": "strongPassword123"
}
```
**Ответ:**
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "9a4f2b1c...",
  "user": {
    "id": 1,
    "email": "developer@company.com",
    "company_name": "Acme Corp",
    "created_at": "2026-10-23T20:30:00Z",
    "updated_at": "2026-10-23T20:30:00Z"
  }
}
```

### 3. Обновление токена (`POST /refresh`)
Токен автоматически подхватывается из HttpOnly-куки `refresh_token`, либо может быть передан в JSON:
```json
{
  "refresh_token": "9a4f2b1c..."
}
```

---

## Запуск в Docker Compose

```bash
docker compose up --build -d
```
Сервис поднимет:
- PostgreSQL на порту `5433` (чтобы не конфликтовать с другими проектами)
- Redis на порту `6380`
- Миграции базы данных
- `gateway` API на порту `8080`
