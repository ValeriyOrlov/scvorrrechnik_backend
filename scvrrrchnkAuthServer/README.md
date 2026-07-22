# Scvrrrchnk Auth Server (Golang JWT)

Сервер аутентификации и авторизации на Go, реализующий регистрацию, вход с выдачей пары JWT‑токенов (Access + Refresh), обновление токенов и выход из системы. 

## Технологический стек

- **Язык:** Go 1.21+
- **Веб-фреймворк:** [Fiber](https://gofiber.io/)
- **База данных:** PostgreSQL
- **ORM:** [GORM](https://gorm.io/)
- **Аутентификация:** JWT ([golang-jwt](https://github.com/golang-jwt/jwt))
- **Хеширование паролей:** [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- **Логирование:** [Logrus](https://github.com/sirupsen/logrus)
- **Миграции:** AutoMigrate (GORM)

## Быстрый старт

### 1. Клонирование репозитория
```bash
git clone https://github.com/ValeriyOrlov/scvrrrchnkAuthServer.git
cd scvrrrchnkAuthServer
```
### 2. Отредактируйте .env, указав свои значения (необходимый перечень переменных представлен в .env.example)

### 3. API-эндпоинты
Все запросы отправляются с заголовком Content-Type: application/json.

Регистрация нового пользователя
POST /register

Тело запроса:
```json
{
  "email": "user@example.com",
  "username": "john",
  "password": "12345678"
}
```

Успешный ответ (201 Created):
```json
{
  "id": 1,
  "email": "user@example.com",
  "username": "john",
  "created_at": "2025-05-03T12:00:00Z",
  "updated_at": "2025-05-03T12:00:00Z"
}
```
Ошибки: 400 (некорректные данные), 409 (пользователь уже существует).

Вход в систему
POST /login

Тело запроса:
```json
{
  "email": "user@example.com",
  "password": "12345678"
}
```
Успешный ответ (200 OK):
```json
{
  "access_token": "eyJhbGciOiJI...",
  "refresh_token": "eyJhbGciOiJI...",
  "token_type": "bearer"
}
```
Ошибки: 401 (неверные учетные данные).

Обновление токенов (ротация)
POST /refresh

Тело запроса:
```json
{
  "refresh_token": "eyJhbGciOiJI..."
}
```
Успешный ответ (200 OK):

```json
{
  "access_token": "eyJhbGciOiJI...",
  "refresh_token": "eyJhbGciOiJI...",
  "token_type": "bearer"
}
```
Ошибки: 401 (невалидный или уже использованный refresh-токен).

Выход из системы
POST /logout

Тело запроса:

```json
{
  "refresh_token": "eyJhbGciOiJI..."
}
```
Успешный ответ (200 OK):

```json
{
  "message": "logged out"
}
```
Защищённый эндпоинт (пример)
GET /me

Заголовок: Authorization: Bearer <access_token>

Успешный ответ (200 OK):

```json
{
  "user_id": 1
}
```
Ошибки: 401 (токен отсутствует, просрочен или неверен).