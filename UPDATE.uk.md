# Оновлення заявок

Гілка: `codex/applications`. Вона включає необхідну Go/Svelte основу, якої ще не було в origin/main. Оновлюються форма підтвердження, копіювання номера, контакт, чернетка та приватна сторінка статусу. Main не змінено.

## Якщо сайт запущено з готового release-пакета

Команди для Linux, з Docker Compose v2 та Git. Приклад припускає, що чинні compose.yaml і .env лежать у `/opt/gradient-monitor`, а Compose project має ім’я `gradient-monitor`. Якщо у вас інші шляхи чи project name, використовуйте саме чинні. Не змінюйте пароль БД або ім’я проєкту.

Спочатку створіть резервну копію поза Git:

```bash
cd /opt/gradient-monitor
umask 077
docker compose -p gradient-monitor exec -T db pg_dump -U uamonitor -d uamonitor -Fc > "backup-$(date +%Y%m%d-%H%M%S).dump"
git clone --branch codex/applications --single-branch https://github.com/xlanyleeet/uamonitor.git /opt/gradient-monitor-src
cd /opt/gradient-monitor-src
```

Якщо цей checkout уже існує, замість clone: `git -C /opt/gradient-monitor-src pull --ff-only origin codex/applications` після перевірки відсутності локальних змін і правильної гілки.

Створіть override з абсолютними шляхами (Compose рахує відносні шляхи від першого compose-файла):

```bash
cat > /opt/gradient-monitor-src/compose.server.yaml <<'YAML'
services:
  backend:
    build:
      context: /opt/gradient-monitor-src/backend-go
  frontend:
    build:
      context: /opt/gradient-monitor-src/frontend
YAML
```

Збережіть старі образи для відкату й зберіть нові. Використовуємо чинний release compose, щоб зберегти мережу, origin і налаштування proxy. Команди нижче припускають теги образів `2026.10.27` із цього release; перед запуском звірте їх із чинним compose.yaml:

```bash
docker image tag gradient-monitor-api:2026.10.27 gradient-monitor-api:before-applications
docker image tag gradient-monitor-web:2026.10.27 gradient-monitor-web:before-applications
docker compose --env-file /opt/gradient-monitor/.env -p gradient-monitor -f /opt/gradient-monitor/compose.yaml -f /opt/gradient-monitor-src/compose.server.yaml build backend frontend
docker compose --env-file /opt/gradient-monitor/.env -p gradient-monitor -f /opt/gradient-monitor/compose.yaml run --rm --no-deps migrate
docker compose --env-file /opt/gradient-monitor/.env -p gradient-monitor -f /opt/gradient-monitor/compose.yaml up -d --no-build --pull never --wait backend frontend proxy
docker compose --env-file /opt/gradient-monitor/.env -p gradient-monitor -f /opt/gradient-monitor/compose.yaml restart proxy
```

## Якщо сайт уже запущено з вихідного коду

У чинному checkout після перевірки `git status`, використовуючи чинний Compose project name:

```bash
git fetch origin
git switch --track origin/codex/applications
docker compose -p uamonitor-go exec -T db pg_dump -U uamonitor -d uamonitor -Fc > "backup-$(date +%Y%m%d-%H%M%S).dump"
docker compose -p uamonitor-go build backend frontend
docker compose -p uamonitor-go run --rm --no-deps migrate
docker compose -p uamonitor-go up -d --no-build --wait backend frontend proxy
docker compose -p uamonitor-go restart proxy
```

Якщо гілка вже існує, використовуйте `git switch codex/applications` і `git pull --ff-only`. Не перемикайте гілки поверх незбережених робочих змін. Базовий docker-compose.yml розрахований на власну інсталяцію; для release-пакета використовуйте попередню секцію.

## Перевірка

```bash
curl -fsS http://127.0.0.1:9013/api/applications/config
curl -fsS -o /dev/null http://127.0.0.1:9013/add-server
```

За налаштованого webhook очікується enabled:true. У браузері перевірте відновлення заповненої чернетки після оновлення, потім надішліть одну службову заявку, перевірте Discord, копіювання номера й повне приватне посилання на статус. Після рішення модератора натисніть «Оновити статус». Не використовуйте реальний сервер для тестового схвалення без наміру додати його в каталог.

Нових міграцій у цьому оновленні немає; для старої інсталяції migrate застосує відсутні початкові міграції. Не запускайте down -v, не замінюйте .env прикладом і не імпортуйте приклад серверів поверх робочого каталогу.

Відкат release після збірки: поверніть теги api/web з `before-applications` на `2026.10.27`, повторіть up з `--force-recreate --no-build --pull never` для backend/frontend і restart proxy. БД не видаляйте. Створений dump містить приватні дані; зберігайте його з обмеженим доступом.
