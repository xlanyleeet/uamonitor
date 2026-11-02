# Ґрадієнт · моніторинг · 2026.10.27

Пакет для Linux x86_64 / amd64 з Docker Engine і Docker Compose v2+. Чотири runtime-образи в `images.tar`: Go backend (також міграції), Svelte frontend, Nginx proxy та PostgreSQL 17. Збірка на сервері не потрібна. Архітектуру перевірте командою `uname -m`: потрібне `x86_64`.

Секретів, локальної `.env`, даних PostgreSQL та приватних контактів у пакеті немає. База створиться порожньою. Доступ до інтернету потрібен для Discord webhook та опитування Minecraft-серверів.

## Запуск

Скопіюйте архів на сервер і розпакуйте у власну директорію, наприклад `/opt/gradient-monitor`. Усі наступні команди запускайте з неї.

```bash
sha256sum -c images.tar.sha256
docker load -i images.tar
cp .env.example .env
chmod 600 .env
openssl rand -hex 32
nano .env
```

Запишіть згенерований hex-пароль у `UM_DB_PASSWORD`, а Discord webhook — у `UM_DISCORD_WEBHOOK_URL`. Залиште `UM_PUBLIC_URL=https://monitoring.gradient-mc.com`. Пароль БД використовується в URL, тому hex безпечний для цього формату. Webhook не публікуйте.

Якщо мережа `172.30.91.0/24` перетинається з вашими Docker/VPN-мережами, змініть `UM_NETWORK_SUBNET` і відповідний `UM_PROXY_GATEWAY` разом. Proxy довіряє клієнтському IP тільки від заданого gateway. Порт 9013 слухає лише loopback хоста.

```bash
docker compose -p gradient-monitor up -d --pull never --wait
docker compose -p gradient-monitor ps
curl -fsS http://127.0.0.1:9013/api/applications/config
```

З webhook очікуйте `{"enabled":true}`. Це підтверджує конфігурацію; реальну доставку перевіряйте окремою тестовою заявкою. Міграції застосовуються автоматично перед запуском backend. Публічна адреса та кнопки Discord потребують налаштованого DNS, TLS і зовнішнього Nginx.

## Cloudflare та зовнішній Nginx

У Cloudflare: DNS A `monitoring` → IP цього сервера, Proxied. SSL/TLS — Full (strict); сертифікат у `/etc/cloudflare/fullchain.pem` має покривати піддомен. Cache Rule для hostname `monitoring.gradient-mc.com` → Bypass cache.

`nginx-origin.conf` — окремий server block для зовнішнього Nginx. Перед його підключенням створіть snippet довірених Cloudflare IP. На Debian/Ubuntu:

```bash
mkdir -p /tmp/gradient-monitor-cf
curl -fsS https://www.cloudflare.com/ips-v4 -o /tmp/gradient-monitor-cf/ipv4
curl -fsS https://www.cloudflare.com/ips-v6 -o /tmp/gradient-monitor-cf/ipv6
awk 'NF { print "set_real_ip_from " $0 ";" }' /tmp/gradient-monitor-cf/ipv4 /tmp/gradient-monitor-cf/ipv6 > /tmp/gradient-monitor-cf/cloudflare-real-ip.conf
printf '\nreal_ip_header CF-Connecting-IP;\n' >> /tmp/gradient-monitor-cf/cloudflare-real-ip.conf
sudo mkdir -p /etc/nginx/snippets
sudo install -m 644 /tmp/gradient-monitor-cf/cloudflare-real-ip.conf /etc/nginx/snippets/cloudflare-real-ip.conf
sudo install -m 644 nginx-origin.conf /etc/nginx/sites-available/gradient-monitor.conf
sudo ln -s /etc/nginx/sites-available/gradient-monitor.conf /etc/nginx/sites-enabled/gradient-monitor.conf
sudo nginx -t && sudo systemctl reload nginx
```

Якщо конфіг з цим server_name вже доданий, оновіть його замість створення дубля. Шляхи sites-available/sites-enabled стосуються Debian/Ubuntu. Списки мереж Cloudflare періодично оновлюйте. Не додавайте `0.0.0.0/0` до довірених proxy. Вбудований Nginx передає IP через другий рівень proxy, а frontend отримує правильний HTTPS origin.

Перевірте `https://monitoring.gradient-mc.com/add-server`, надішліть службову заявку, перевірте повідомлення в Discord і відкриття кнопок. GET не додає сервер; остаточне рішення потребує POST. Приватні посилання не пересилайте.

## Приклад Ґрадієнту

Лише для нової порожньої БД можна імпортувати включений приклад; JSON-імпорт оновлює записи з відповідними ID:

```bash
cat servers.example.json | docker compose -p gradient-monitor exec -T backend sh -c 'cat > /tmp/servers.json && uamonitor -import /tmp/servers.json'
docker compose -p gradient-monitor restart backend
```

## Дані та оновлення

PostgreSQL зберігає дані у volume проєкту `gradient-monitor`. Зупинка: `docker compose -p gradient-monitor down`. Зберігайте те саме ім’я проєкту при оновленнях. Перед оновленням зробіть backup:

```bash
docker compose -p gradient-monitor exec -T db pg_dump -U uamonitor -d uamonitor > backup.sql
```

Не використовуйте `down -v` для робочої інсталяції: це видаляє БД. Зміна пароля в `.env` не змінює пароль уже створеної БД. Секрети, backup і приватні посилання мають залишатися доступними лише адміністраторам.

Налаштування заявок і обмеження описано в `ADMISSION.uk.md`. Кнопки звичайного webhook відкривають приватну сторінку сайту; Discord OAuth та автоматичні DM власникам не реалізовані.
