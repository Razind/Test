# Simple Go URL Shortener (Microservices)

Минимальный пример сервиса сокращения ссылок в стиле click.yandex с микросервисной архитектурой.

## Сервисы

- **storage**: хранение кодов и URL (in-memory).
- **api**: генерация коротких кодов и запись в storage.
- **redirect**: редирект по короткому коду, получает данные из storage.

## Запуск

Откройте три терминала и запустите:

```bash
go run ./cmd/storage
```

```bash
go run ./cmd/api
```

```bash
go run ./cmd/redirect
```

## Запуск через Docker (одной командой)

Если установлен Docker, можно поднять все сервисы **вместе с PostgreSQL** одной командой:

```bash
docker compose up --build
```

Compose поднимет контейнер `db` с тестовыми доступами и автоматически применит схему из `sql/schema.sql` при первом старте тома.

Тестовые доступы к БД:

- user: `shortener`
- password: `shortener`
- database: `shortener`
- host внутри docker-сети: `db:5432`
- host с машины: `localhost:5432`

После запуска сервисы будут доступны на тех же портах:

- UI/API: http://localhost:8080
- Redirect: http://localhost:8081

## PostgreSQL

В `docker-compose.yml` PostgreSQL уже настроен и подключается к `storage` автоматически через:

```bash
postgres://shortener:shortener@db:5432/shortener?sslmode=disable
```

Если запускаете `storage` отдельно (без compose), задайте `DATABASE_URL` вручную:

```bash
export DATABASE_URL="postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable"
```

Схема базы находится в `sql/schema.sql` (в compose применяется автоматически при первом старте БД-тома).

## Пример использования

Откройте в браузере `http://localhost:8080` и используйте веб-интерфейс.

В веб-интерфейсе доступны регистрация, авторизация, список ссылок, их удаление и статистика переходов.

## Основные возможности

- Регистрация и авторизация по email/паролю.
- Профиль с суммарной статистикой переходов и количеством ссылок.
- Список ссылок текущего пользователя, удаление ссылок.
- Счетчик переходов обновляется при переходе по короткой ссылке.

## Примеры API-запросов

Регистрация:

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

Логин:

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'
```

Создание короткой ссылки (нужна активная сессия):

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'
```

Получение списка ссылок:

```bash
curl http://localhost:8080/links
```

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'
```

В ответе вернется `short_url` вида `http://localhost:8081/<code>`. Откройте его в браузере, и вы получите редирект на исходный URL.

## Ограничения

- Если `DATABASE_URL` не задан, все данные хранятся в памяти (in-memory) и при перезапуске сервисов сбрасываются.
- Сессии авторизации также не сохраняются между перезапусками.

## Переменные окружения

| Сервис   | Переменная   | Значение по умолчанию |
|----------|--------------|-----------------------|
| api      | `PORT`       | `8080`                |
| api      | `STORAGE_URL`| `http://localhost:8082` |
| api      | `BASE_URL`   | `http://localhost:8081` |
| redirect | `PORT`       | `8081`                |
| redirect | `STORAGE_URL`| `http://localhost:8082` |
| storage  | `PORT`       | `8082`                |
| storage  | `DATABASE_URL` | пусто (in-memory) |
