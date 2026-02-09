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

- Все данные хранятся в памяти (in-memory), при перезапуске сервисов они сбрасываются.
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
