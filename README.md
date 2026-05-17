# GiftScheduleBot

Telegram-бот для отслеживания розыгрышей. Пользователи сохраняют посты с розыгрышами (форвард или ссылка), бот парсит дату окончания, отправляет уведомления за 3 дня / 1 день / 1 час и при завершении, а также проверяет — победил ли пользователь.

---

## Функционал

- **Добавление розыгрышей** — форвард из канала или ссылка `t.me/channel/123`
- **Парсинг** — название, призы, дата окончания (из текста поста)
- **Уведомления** — за 3 дня, 1 день, 1 час и при завершении (настраиваемые)
- **Проверка победителя** — по username / имени в тексте итогов
- **Мои розыгрыши** — список с пагинацией, удаление
- **Общий список** — все розыгрыши от всех пользователей (10 на страницу)
- **Inline-режим** — `@bot` в любом чате → поделиться последними 5 розыгрышами
- **Профиль** — статистика и настройки уведомлений
- **Админ-панель** — `/admin`: статистика, управление пользователями (бан), рассылка, управление постами

---

## Структура проекта

```
GiftScheduleBot/
├── main.go
├── go.mod
├── .env.example
├── config/config.go        — загрузка переменных окружения
├── db/
│   ├── db.go               — подключение и миграции SQLite
│   ├── users.go            — CRUD пользователей
│   └── posts.go            — CRUD постов и user_posts
├── bot/
│   ├── bot.go              — инициализация, роутинг, хелперы
│   ├── handlers.go         — /start, текст, форвард, ссылка
│   ├── callbacks.go        — обработка inline-кнопок
│   ├── inline.go           — inline query (@bot в чате)
│   ├── scheduler.go        — cron-уведомления
│   └── admin.go            — /admin панель
└── parser/
    ├── giveaway.go         — парсинг поста (ключевые слова, дата, призы)
    └── results.go          — проверка победителя в итоговом посте
```

---

## Переменные окружения

Скопировать `.env.example` → `.env` и заполнить:

```env
BOT_TOKEN=your_bot_token_here       # токен от @BotFather
DB_PATH=./giftbot.db                # путь к SQLite файлу
DUMP_CHAT_ID=-1001234567890         # ID чата/канала где бот — администратор
                                    # используется для чтения содержимого постов по ссылке
ADMIN_IDS=123456789,987654321       # Telegram ID администраторов через запятую
```

> **DUMP_CHAT_ID** — создать приватный канал, добавить туда бота как администратора, скопировать ID (можно узнать через @userinfobot или переслав сообщение из канала в @RawDataBot).

---

## Запуск

### Требования

| Платформа | Требование |
|---|---|
| Все | Go 1.22+ |
| Все | Telegram Bot Token |

---

### Windows

**1. Установить Go**

Скачать установщик с [go.dev/dl](https://go.dev/dl/) и запустить. После установки перезапустить терминал.

```powershell
go version
```

**2. Клонировать репозиторий**

```powershell
git clone https://github.com/N1k3YB/GiftScheduleBot.git
cd GiftScheduleBot
```

**3. Создать `.env`**

```powershell
copy .env.example .env
notepad .env
```

Заполнить `BOT_TOKEN`, `DUMP_CHAT_ID`, `ADMIN_IDS`.

**4. Загрузить зависимости**

```powershell
go mod tidy
```

**5. Запустить**

```powershell
go run .
```

Или собрать бинарник:

```powershell
go build -o bin\giftbot.exe .
.\bin\giftbot.exe
```

**Автозапуск через Task Scheduler (опционально)**

Создать задачу в «Планировщике задач» → триггер «При запуске» → действие: запуск `giftbot.exe`.

---

### Linux

**1. Установить Go**

```bash
# Ubuntu / Debian
sudo apt update && sudo apt install -y golang-go

# Или вручную — актуальную версию взять на go.dev/dl:
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

Проверить:

```bash
go version
```

**2. Клонировать репозиторий**

```bash
git clone https://github.com/N1k3YB/GiftScheduleBot.git
cd GiftScheduleBot
```

**3. Создать `.env`**

```bash
cp .env.example .env
nano .env   # или vim .env
```

**4. Загрузить зависимости и запустить**

```bash
go mod tidy
go run .
```

Или собрать бинарник:

```bash
go build -o bin/giftbot .
./bin/giftbot
```

**Автозапуск через systemd**

Создать файл `/etc/systemd/system/giftbot.service`:

```ini
[Unit]
Description=GiftScheduleBot
After=network.target

[Service]
Type=simple
User=YOUR_USER
WorkingDirectory=/path/to/GiftScheduleBot
ExecStart=/path/to/GiftScheduleBot/bin/giftbot
Restart=on-failure
RestartSec=5s
EnvironmentFile=/path/to/GiftScheduleBot/.env

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable giftbot
sudo systemctl start giftbot
sudo systemctl status giftbot
```

Логи:

```bash
journalctl -u giftbot -f
```

**Автозапуск через PM2 (альтернатива systemd)**

```bash
# Установить PM2 (нужен Node.js)
npm install -g pm2

# Собрать бинарник и запустить через PM2
go build -o bin/giftbot .
pm2 start ./bin/giftbot --name giftbot

# Сохранить список процессов и добавить в автозапуск
pm2 save
pm2 startup
# Выполнить команду, которую выведет pm2 startup
```

Полезные команды PM2:

```bash
pm2 status          # статус процессов
pm2 logs giftbot    # логи в реальном времени
pm2 restart giftbot # перезапуск
pm2 stop giftbot    # остановка
```

---

### macOS

**1. Установить Go**

```bash
# Через Homebrew
brew install go

# Или скачать установщик с go.dev/dl
```

Проверить:

```bash
go version
```

**2. Клонировать репозиторий**

```bash
git clone https://github.com/N1k3YB/GiftScheduleBot.git
cd GiftScheduleBot
```

**3. Создать `.env`**

```bash
cp .env.example .env
open -e .env   # откроет в TextEdit
# или
nano .env
```

**4. Загрузить зависимости и запустить**

```bash
go mod tidy
go run .
```

Или собрать бинарник:

```bash
go build -o bin/giftbot .
./bin/giftbot
```

**Автозапуск через launchd**

Создать `~/Library/LaunchAgents/com.giftbot.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.giftbot</string>
  <key>ProgramArguments</key>
  <array>
    <string>/path/to/GiftScheduleBot/bin/giftbot</string>
  </array>
  <key>WorkingDirectory</key>
  <string>/path/to/GiftScheduleBot</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/giftbot.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/giftbot.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.giftbot.plist
launchctl start com.giftbot
```

**Автозапуск через PM2 (альтернатива launchd)**

```bash
# Установить PM2 (нужен Node.js / Homebrew: brew install node)
npm install -g pm2

# Собрать бинарник и запустить через PM2
go build -o bin/giftbot .
pm2 start ./bin/giftbot --name giftbot

# Сохранить и добавить в автозапуск
pm2 save
pm2 startup
# Выполнить команду, которую выведет pm2 startup
```

Полезные команды PM2:

```bash
pm2 status          # статус процессов
pm2 logs giftbot    # логи в реальном времени
pm2 restart giftbot # перезапуск
pm2 stop giftbot    # остановка
```

---

## Использование бота

| Действие | Как |
|---|---|
| Добавить розыгрыш | Переслать пост из канала или отправить ссылку `t.me/channel/123` |
| Мои розыгрыши | Кнопка «📋 Мои розыгрыши» в главном меню |
| Все розыгрыши | Кнопка «🌐 Все розыгрыши» |
| Поделиться | Написать `@botusername` в любом чате |
| Профиль и уведомления | Кнопка «👤 Профиль» |
| Удалить розыгрыш | Кнопка «🗑» в списке |
| Проверить результаты | Кнопка «🔍 #ID» у завершённого розыгрыша |
| Админ-панель | `/admin` (только для ADMIN_IDS) |

---

## Лицензия

MIT
