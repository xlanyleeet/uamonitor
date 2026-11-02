# План міграції UAMonitor

Окрема копія: `C:\Users\xlanyleeet\Documents\GitHub\uamonitor-go`.
Оригінальний репозиторій не змінюється. Старий backend залишається для порівняння API.

1. Зберегти копію з історією Git і поточними змінами.
2. Реалізувати Go API зі збереженням `/api/servers`, `/api/servers/:id`, `/api/graph`, `/api/graph/:id` та JSON DTO.
3. Додати PostgreSQL, версійовані транзакційні міграції, індекси й імпорт каталогу серверів.
4. Реалізувати Minecraft Java status ping: DNS/SRV, обмеження пакетів, timeout, перевірка мережевих адрес, обмежений пул, distributed lock, graceful shutdown.
5. Виправити запити графіків, кеш, rate limit, retention і розділити мережеві та DB-помилки.
6. Оновити frontend tooling, прибрати N+1, виправити API base, типи, modal/embed, описи, графіки й listeners.
7. Зібрати Docker Compose: PostgreSQL → migrate → backend → frontend → reverse proxy. Додати volumes, healthchecks і непривілейований runtime.
8. Перевірити Go unit/race/integration tests, frontend check/build/audit, Docker build і HTTP smoke test, persistence після restart.
9. Записати фактичні результати й обмеження в `VALIDATION.uk.md`; додати інструкцію запуску та backup/restore.

Дані з SQLite не видаляються. Автоматичний перенос історії потребує наявної справної SQLite-схеми; у вихідній робочій копії запит до Server завершувався відсутністю таблиці.

Усі дев'ять етапів виконано. Фактичні перевірки та межі результату: [VALIDATION.uk.md](VALIDATION.uk.md).
