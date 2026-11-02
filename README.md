# UAMonitor — Go + PostgreSQL

Моніторинг Minecraft Java-серверів. Окрема модернізована копія початкового UAMonitor.
Дизайн адаптовано з ArtList.dev: [опис перенесення](DESIGN.uk.md), [MIT attribution](THIRD_PARTY_NOTICES.md).
План робіт: [PLAN.uk.md](PLAN.uk.md). Початковий аудит: [AUDIT.uk.md](AUDIT.uk.md).

## Запуск через Docker

Потрібен Docker Compose v2. З кореня цієї копії:

```powershell
Copy-Item .env.example .env
# Вкажіть у .env власний довгий випадковий URL-safe UM_DB_PASSWORD.
docker compose -p uamonitor-go up -d --build
node deploy/smoke.mjs
```

Повна локальна перевірка заповненого API/embed і persistence: `node deploy/validate-stack.mjs`. Вона тимчасово додає службовий запис, перезапускає PostgreSQL/backend цього stack і прибирає запис у `finally`; запускайте її лише на локальному тестовому deployment.

На цій машині `.env` вже створено з випадковим паролем. Не перезаписуйте його після створення PostgreSQL volume: зміна env не змінює пароль уже створеної БД.

Сайт: http://127.0.0.1:9013. Порт змінюється через `UM_HTTP_PORT`.
Відбір: `/requirements` — критерії, `/add-server` — заявка з ручною модерацією через Discord. Без налаштованого webhook надсилання вимкнено. Інструкція: [ADMISSION.uk.md](ADMISSION.uk.md).
PostgreSQL і backend не публікують порти на хості. Контейнери мають окремий project name `uamonitor-go`; сервіси магазину не змінюються.

Compose запускає PostgreSQL, чекає його healthcheck, виконує міграції, запускає backend і frontend, потім reverse proxy.
Go працює як непривілейований користувач; runtime images не містять вихідного коду чи `.env`.
Frontend використовує Node 24, SvelteKit 3, Svelte 5. PostgreSQL 17 зберігає дані в named volume.

Для VPS залиште bind `127.0.0.1` і проксіюйте через наявний HTTPS reverse proxy. Якщо він термінує TLS перед цим nginx, налаштуйте коректний зовнішній origin для SvelteKit, наприклад `ORIGIN=https://monitor.example.org` у frontend environment. Публікувати БД не потрібно. `UM_TRUST_PROXY=true` у Compose припускає, що API доступне тільки через приватну Docker-мережу; вимкніть це при прямій публікації backend.

## Додавання серверів

Каталог порожній за замовчуванням. Реальні сервери автоматично не додаються.
Файл `backend-go/servers.example.json` містить готовий приклад Ґрадієнту (`gg.gradient-mc.com:25565`) з описом і посиланнями. У поточній локальній інсталяції цей сервер уже імпортовано під ID 1.
Створіть `backend-go/servers.local.json` на основі `backend-go/servers.example.json`. Вкажіть реальні hostname, порт, назву, явні додатні ID, за потреби опис і HTTP(S)-посилання. `changeIcon` задає оновлення PNG-іконки з відповіді Minecraft.

```powershell
docker compose -p uamonitor-go run --rm -v "${PWD}/backend-go/servers.local.json:/import/servers.json:ro" backend -import /import/servers.json
docker compose -p uamonitor-go restart backend
```

Імпорт транзакційний: оновлює сервери за ID, не видаляє інші й не перезаписує історію. JSON можна експортувати зі старого `/api/servers`; адреси повинні бути hostname/IP без `host:port`, порт — окремий рядок. Значення YouTube — повний HTTP(S) URL. Не завантажуйте сторонні описи без перевірки: frontend очищає Markdown дозволеним списком HTML.

Стара папка `backend/`, її `.env` та SQLite залишені як архів і не використовуються новим Compose. Автоматичного destructive reset або імпорту старої історії немає. Початкова SQLite не мала таблиці Server, тому дані каталогу з неї не переносилися.

## Налаштування

| Змінна | Типове значення | Призначення |
|---|---|---|
| `UM_DB_PASSWORD` | обов'язкова | URL-safe пароль PostgreSQL |
| `UM_BIND` | `127.0.0.1` | bind вебпорту |
| `UM_HTTP_PORT` | `9013` | вебпорт |
| `UM_POLL_SECONDS` | `600` | інтервал опитування, мінімум 10 с |
| `UM_POLL_CONCURRENCY` | `16` | кількість паралельних ping, 1–64 |
| `UM_RETENTION_DAYS` | `30` | термін збереження історії |

Backend також підтримує `UM_DATABASE_URL`, `UM_LISTEN`, `UM_TRUST_PROXY` та `UM_ALLOW_PRIVATE_TARGETS`. Останній за замовчуванням false: приватні, loopback, link-local та службові адреси заборонено, включно з результатами SRV/DNS. Не вмикайте його для недовірених каталогів. Додатковий egress firewall рекомендований для сервісів, які опитують сторонні сервери.

API зберігає поточні GET-шляхи `/api/servers`, `/api/servers/:id`, `/api/graph`, `/api/graph/:id`.
Порт у JSON лишається рядком. Відсутні значення — null, відсутні спостереження — порожній масив.
Некоректний ID дає 400, відсутній сервер — 404. `/health` перевіряє процес, `/ready` — доступ до мігрованої БД.
Зведення API кешуються на 30 секунд. Reverse proxy обмежує частоту запитів за IP; backend має додатковий обмежений за пам'яттю limiter. Для дуже великих каталогів потрібні пагінація і окреме вимірювання навантаження.

Опитування виконується відразу після старту, потім за інтервалом. Пул обмежений, PostgreSQL advisory lock не дозволяє одночасний tick кільком реплікам. Повільна відповідь не довше 5 секунд. Відповіді Minecraft обмежено за розміром; невалідні іконки ігноруються. Помилки БД не підмінюються статусом offline. Закриття процесу скасовує опитування й завершує HTTP-запити. Старі статуси видаляються пакетами; останнє спостереження сервера зберігається для показу stale.

## Перевірки

```powershell
cd backend-go
go test ./...
go vet ./...
cd ../frontend
npm ci
npm run check
npm run build
npm test
npm audit
cd ..
docker compose -p uamonitor-go -f docker-compose.yml -f compose.test.yaml --profile test run --rm --build test
node deploy/smoke.mjs
```

Для локального frontend потрібен Node >=24; Go >=1.26.10. Інтеграційні Go-тести запускаються з `UM_TEST_DATABASE_URL`, створюють у тестовій БД унікальну схему й видаляють лише її. Через Docker це окрема тестова схема тієї самої локальної інсталяції. Тести використовують власний локальний Minecraft fixture, не зовнішні сервери.
`npm test` перевіряє зібраний SSR frontend із контрольованим API; спочатку виконайте build.

## Резервне копіювання та відновлення

Формуйте backup через pg_dump, а не копіювання активних PostgreSQL-файлів. Windows-команди нижче не передають бінарний дамп через PowerShell pipe:

```powershell
docker compose -p uamonitor-go exec -T db pg_dump -U uamonitor -d uamonitor -Fc -f /tmp/uamonitor.dump
docker compose -p uamonitor-go cp db:/tmp/uamonitor.dump ./uamonitor.dump
```

Зберігайте дамп поза репозиторієм. Перевіряйте відновлення в окремій БД. Приклад для НОВОЇ порожньої БД:

```powershell
docker compose -p uamonitor-go cp ./uamonitor.dump db:/tmp/restore.dump
docker compose -p uamonitor-go exec -T db createdb -U uamonitor uamonitor_restore
docker compose -p uamonitor-go exec -T db pg_restore -U uamonitor -d uamonitor_restore --no-owner --no-privileges /tmp/restore.dump
```

Зупинити без втрати даних: `docker compose -p uamonitor-go down`. Не додавайте `-v`, якщо хочете зберегти БД.
