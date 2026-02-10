# Postgres в Docker (готово к запуску)

Сделал готовую конфигурацию, чтобы база запускалась сразу в контейнере с тестовыми доступами.

## Что внутри

- `postgres` (PostgreSQL 16)
- `adminer` (web-интерфейс для просмотра БД)
- автоинициализация таблицы `users` и тестовых записей

## Быстрый старт

```bash
docker compose up -d
```

## Доступы к БД

Берутся из `.env`:

- DB: `app_db`
- User: `app_user`
- Password: `app_password`
- Host: `localhost`
- Port: `5432`

## Adminer

После запуска:

- URL: <http://localhost:8080>
- System: `PostgreSQL`
- Server: `postgres`
- Username: `app_user`
- Password: `app_password`
- Database: `app_db`

## Проверка, что всё работает

```bash
docker compose ps
docker compose exec postgres psql -U app_user -d app_db -c "SELECT * FROM users;"
```
