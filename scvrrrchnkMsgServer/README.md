# Scvrrrchnk Messenger Server (Go)

Серверная часть мессенджера реального времени, реализованная на Go. Поддерживает создание чатов, отправку сообщений по REST и мгновенную доставку через WebSocket. Использует отдельный [сервер авторизации](https://github.com/ValeriyOrlov/scvrrrchnkAuthServer) для регистрации и аутентификации по JWT.

## Технологический стек

- **Язык:** Go 1.21+
- **Веб-фреймворк:** [Fiber v2](https://gofiber.io/)
- **WebSocket:** [gofiber/contrib/websocket](https://github.com/gofiber/contrib/websocket)
- **База данных:** PostgreSQL
- **ORM:** [GORM](https://gorm.io/)
- **Аутентификация:** JWT ([golang-jwt](https://github.com/golang-jwt/jwt))
- **Логирование:** [Logrus](https://github.com/sirupsen/logrus)
- **Миграции:** GORM AutoMigrate

## Быстрый старт

### 1. Клонирование репозитория
```bash
git clone https://github.com/ValeriyOrlov/scvrrrchnkMsgServer.git
cd scvrrrchnkMsgServer
```

### 2. Настройка переменных окружения
Создайте в корне файл .env по примеру .env.example:

```bash
cp .env.example .env
```
Отредактируйте .env, указав свои значения:
```
text
APP_PORT=8081
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=messenger
JWT_SECRET=super-secret-key
```
JWT_SECRET должен совпадать с ключом, используемым сервером авторизации.

### 3. Запуск базы данных
Убедитесь, что PostgreSQL запущен и создана база данных messenger. При первом запуске GORM автоматически создаст все таблицы.

4. Запуск сервера
```bash
go run ./cmd/server
```
Сервер стартует на порту, указанном в APP_PORT (по умолчанию 8081).

API-эндпоинты
Защищённые маршруты (требуют JWT)
Все эндпоинты, кроме WebSocket, требуют заголовок Authorization: Bearer <access_token>.
При первом обращении пользователя к любому защищённому маршруту мессенджер автоматически синхронизирует его данные с сервером авторизации.



POST	/api/chats	Создать чат	{"type": "group", "chat_name": "Friends", "member_ids": [2,3]}	201 Created, объект чата с участниками

GET	/api/chats	Получить список чатов пользователя	—	200 OK, массив чатов с участниками

GET	/api/chats/:id	Получить информацию о чате по ID	—	200 OK, объект чата с участниками

POST	/api/chats/:id/messages	Отправить сообщение в чат	{"content": "Hello!"}	201 Created, объект сообщения с автором

GET	/api/chats/:id/messages?limit=50&offset=0	Получить историю сообщений	—	200 OK, массив сообщений с авторами

WebSocket
Подключение: ws://localhost:8081/ws?token=<JWT>
После установки соединения можно отправлять сообщения в формате JSON:

```json
{
  "type": "chat_message",
  "chat_id": 1,
  "content": "Привет!"
}```
Сервер рассылает новое сообщение всем участникам чата (кроме отправителя), которые находятся онлайн. Ответ приходит в виде полного объекта сообщения с отправителем.