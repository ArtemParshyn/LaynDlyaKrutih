# CashFlow Forecast

Сервис прогнозирования денежных потоков малого бизнеса. Пользователь загружает банковскую выписку, после чего система строит фактическую историю, прогноз остатка, риски кассового разрыва, категории операций и календарь регулярных платежей.

## Возможности

- изоляция данных по отдельным пульсам;
- загрузка CSV-выписки;
- восстановление фактического остатка по `balance_eod`;
- переобучение Prophet на каждой новой выписке;
- прогноз остатка на 1–365 дней;
- кассовый разрыв с датой, суммой, вероятностью и обязательным платежом;
- категоризация операций через FastAPI ML-сервис;
- выявление регулярных списаний и календарь обязательных платежей;
- режим «Что если» с отдельной временной моделью;
- сценарии не изменяют основную выписку и базовую модель.

## Архитектура

```text
Frontend (React)
        |
        v
Gateway (Go) ---- Planning (планы пользователя)
        |
        +---- Ledger (выписка, история, сохранённый прогноз)
        |
        +---- ProphetFastAPI (обучение, прогноз, категории, риски)
```

### Сервисы

| Каталог | Назначение |
|---|---|
| `frontend/` | интерфейс и график |
| `gateway/` | авторизация, пульсы и orchestration |
| `ledger/` | операции, выписки, фактическая история |
| `planning/` | пользовательские будущие события |
| `prophet/` | FastAPI, Prophet, категоризация и аналитика |
| `deploy/` | Docker Compose для сервера |

## CSV-выписки

Основной формат ML-сервиса — UTF-8 CSV с разделителем `;`:

```csv
company_id;operation_id;operation_date;amount;direction;counterparty_inn;counterparty_name;payment_purpose;balance_eod
C075;C075-000001;2024-10-01;1990.00;Списание;0068275202;ПАО «Банк»;Комиссия;761899.18
```

Обязательные поля для обучения:

- `operation_id`;
- `operation_date`;
- `amount`;
- `direction`;
- `balance_eod`.

Для категоризации также нужны `counterparty_inn`, `counterparty_name` и `payment_purpose`.

Сумма в CSV положительная. Направление задаётся полем `direction`:

- `Поступление` — приход;
- любое другое значение — списание.

`balance_eod` — фактический банковский остаток на конец дня. Он используется для обучения Prophet и построения исторической части графика. Суммировать операции от нуля нельзя: это даёт неправильный остаток.

## Обычный прогноз

1. Frontend отправляет CSV в Gateway.
2. Gateway сохраняет выписку в Ledger.
3. Gateway отправляет исходный CSV целиком в `POST /train` FastAPI.
4. FastAPI возвращает `job_id` и обучает модель асинхронно.
5. Gateway опрашивает `GET /jobs/{job_id}`.
6. После `succeeded` Gateway получает `GET /forecast/{company_id}`.
7. Прогноз, риски и историю Gateway сохраняет в Ledger.

Важно: ML-модель обучается по истории остатка `balance_eod`. Категоризация и регулярные платежи являются отдельным аналитическим слоем и не являются признаками Prophet.

## Будущая операция и «Что если»

### Обычная будущая операция

Будущая операция сохраняется в Planning как пользовательский план. Planning хранит событие, но документированный FastAPI `/train` сам по себе планы не принимает: он обучается только по CSV. Поэтому для влияния будущего события на основной прогноз его нужно включить в отдельный временный датасет перед обучением.

### Режим «Что если»

Сценарий не меняет Ledger, Planning или базовую модель.

При выборе операции на будущую дату Gateway:

1. берёт реальные операции из выписки;
2. берёт базовый прогноз до даты вмешательства;
3. превращает прогнозные дневные изменения во временные движения;
4. добавляет пользовательскую операцию в выбранную дату;
5. создаёт временную модель с ID `scenario-*`;
6. отправляет временный CSV в `/train`;
7. ждёт `succeeded` через `/jobs/{job_id}`;
8. получает новый прогноз через `/forecast/{scenario_id}`;
9. удаляет временную модель.

Итоговый график строится как:

```text
факт из выписки → базовый прогноз до операции → операция → новый прогноз Prophet
```

Таким образом, сценарий — это настоящее переобучение на новом временном датасете, а не простой сдвиг линии.

## FastAPI ML API

Источник ML-сервиса: [t3plov/ProphetFastAPI](https://github.com/t3plov/ProphetFastAPI).

Основные endpoint’ы:

| Метод | Endpoint | Назначение |
|---|---|---|
| `GET` | `/health` | состояние ML-сервиса |
| `POST` | `/train` | асинхронное обучение по CSV |
| `GET` | `/jobs/{job_id}` | статус обучения |
| `GET` | `/forecast/{company_id}` | прогноз и `cash_gap` |
| `POST` | `/categorize` | разовая категоризация CSV |
| `GET` | `/analytics/{id}/categories` | сводка категорий |
| `GET` | `/analytics/{id}/operations` | размеченные операции |
| `GET` | `/analytics/{id}/recurring` | регулярные списания |
| `GET` | `/analytics/{id}/calendar` | календарь обязательных платежей |
| `GET` | `/models/{id}` | паспорт модели |
| `DELETE` | `/models/{id}` | удаление модели |

### Риск кассового разрыва

Новая ML-версия считает риск не только по отрицательному остатку. Она сравнивает прогноз с обязательными платежами календаря.

Возможные причины:

- `payment_shortfall` — не хватает денег на обязательный платёж;
- `negative_balance` — прогнозный остаток отрицательный.

Если `cash_gap.risk == false`, это не предупреждение. Gateway не должен показывать такой объект как alert.

## API Gateway

Пульс-ориентированные endpoint’ы:

```text
GET  /api/pulses
POST /api/pulses
GET  /api/pulses/:pulseID/forecast
POST /api/pulses/:pulseID/statements
POST /api/pulses/:pulseID/forecast/process
POST /api/pulses/:pulseID/operations
POST /api/pulses/:pulseID/scenarios
```

Сценарный запрос отправляется на:

```http
POST /api/pulses/{pulseID}/scenarios
```

Тело содержит `events` и/или изменяемые `operations`.

## Локальный запуск

Требования:

- Docker Desktop;
- Git.

```powershell
git clone https://github.com/ArtemParshyn/LaynDlyaKrutih.git
cd LaynDlyaKrutih
docker compose -f deploy/compose.yaml up -d --build
```

Адреса:

- Frontend: `http://localhost:3000`;
- Gateway: `http://localhost:8080`;
- FastAPI доступен внутри Docker-сети на `review-forecast:8000`.

Проверка контейнеров:

```powershell
docker compose -f deploy/compose.yaml ps
docker compose -f deploy/compose.yaml logs -f forecast
```

## Деплой

На сервере используется каталог:

```text
/opt/cashflow/review-prophet
```

Обновление:

```bash
git fetch origin feat/pulses
git merge --ff-only FETCH_HEAD
docker compose -f deploy/compose.yaml build --pull=false
docker compose -f deploy/compose.yaml up -d --no-deps --pull never gateway frontend ledger forecast
```

После изменений frontend очистите кеш браузера через `Ctrl+F5`.

## Хранение данных

- Ledger хранит операции, фактическую историю и сохранённый базовый прогноз.
- Planning хранит будущие пользовательские события.
- FastAPI хранит модели и аналитические файлы в `artifacts/`.
- Сценарные модели удаляются после расчёта.

## Важные правила

- Не использовать сумму операций от нулевого остатка, если в CSV есть `balance_eod`.
- Не считать `cash_gap` риском без проверки `risk == true`.
- Не отправлять FastAPI урезанный CSV, если нужна категоризация: необходимы данные контрагента и назначение платежа.
- Не сохранять сценарные операции в основной Ledger.
- Не смешивать базовый прогноз и сценарный прогноз как две независимые линии: сценарий должен заменять прогноз после пересчёта.

## Репозиторий

https://github.com/ArtemParshyn/LaynDlyaKrutih
