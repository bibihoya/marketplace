# Ручная демонстрация E2E

Демонстрация выполняется вручную после запуска проекта через `make up`. User Service доступен на `localhost:8081`, Commerce Service — на `localhost:8080`. Для административных операций перед запуском нужно скопировать `.env.example` в `.env` и заменить пример пароля.

## Основной сценарий

1. Проверить health-check обоих сервисов.
2. Зарегистрировать продавца и покупателя, затем выполнить вход.
3. Обновить пару токенов и убедиться, что старый refresh-токен больше не принимается.
4. От имени продавца создать несколько товаров, получить список с пагинацией и фильтрами, обновить товар и проверить мягкое удаление.
5. От имени продавца или администратора создать глобальный промокод.
6. От имени покупателя создать заказ с несколькими позициями и промокодом. Проверить итоговую сумму, скидку, снимки цен и уменьшение остатков.
7. Изменить состав заказа и проверить пересчёт суммы и остатков.
8. Отменить заказ и проверить возврат остатков и использования промокода.
9. От имени администратора провести отдельный заказ по цепочке `CREATED → PAYMENT_PENDING → PAID → SHIPPED → COMPLETED`.
10. Показать JSON-логи, заголовок `X-Request-Id` и содержимое таблиц обеих баз.

## Негативные сценарии

- запрос без JWT и с повреждённым JWT;
- попытка `USER` создать товар и попытка `SELLER` создать заказ;
- изменение продавцом чужого товара;
- невалидные поля товара и заказа;
- отсутствующий или неактивный товар;
- недостаточный остаток;
- повторный активный заказ и превышение ограничения частоты;
- недействительный, истёкший или неприменимый промокод;
- недопустимый переход состояния заказа;
- попытка получить чужой заказ.

Для каждого случая проверяются HTTP-статус, `error_code`, `message` и `details`. После ошибки нужно убедиться, что транзакционные изменения не сохранились.

## Основные команды

Для примеров ниже нужен `jq`. Пароли используются только в локальном окружении.

```bash
USER_API=http://localhost:8081
COMMERCE_API=http://localhost:8080

curl -i "$USER_API/health"
curl -i "$COMMERCE_API/health"

curl -sS -X POST "$USER_API/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"seller@example.com","password":"seller-password","role":"SELLER"}' | jq

curl -sS -X POST "$USER_API/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"buyer@example.com","password":"buyer-password","role":"USER"}' | jq

SELLER_LOGIN="$(curl -sS -X POST "$USER_API/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"seller@example.com","password":"seller-password"}')"
SELLER_TOKEN="$(printf '%s' "$SELLER_LOGIN" | jq -r '.access_token')"

BUYER_LOGIN="$(curl -sS -X POST "$USER_API/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"buyer@example.com","password":"buyer-password"}')"
BUYER_TOKEN="$(printf '%s' "$BUYER_LOGIN" | jq -r '.access_token')"
BUYER_REFRESH="$(printf '%s' "$BUYER_LOGIN" | jq -r '.refresh_token')"

BUYER_REFRESHED="$(curl -sS -X POST "$USER_API/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$BUYER_REFRESH\"}")"
printf '%s' "$BUYER_REFRESHED" | jq
BUYER_TOKEN="$(printf '%s' "$BUYER_REFRESHED" | jq -r '.access_token')"

# Старый refresh-токен после ротации должен вернуть REFRESH_TOKEN_INVALID.
curl -sS -X POST "$USER_API/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$BUYER_REFRESH\"}" | jq

PRODUCT_ID="$(curl -sS -X POST "$COMMERCE_API/products" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Кофемашина","description":"Демонстрационный товар","price":1000,"stock":10,"category":"HOME","status":"ACTIVE"}' \
  | jq -r '.id')"

curl -sS "$COMMERCE_API/products?page=0&size=20&status=ACTIVE&category=HOME" \
  -H "Authorization: Bearer $BUYER_TOKEN" | jq

curl -sS -X POST "$COMMERCE_API/orders" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_id\":\"$PRODUCT_ID\",\"quantity\":2}]}" | jq
```

Состояние контейнеров и логи:

```bash
make ps
make logs
```

`make logs` продолжает следить за логами; выйти можно через `Ctrl+C`.

Проверка данных Commerce DB:

```bash
docker compose exec commerce-db psql -U marketplace -d marketplace_commerce \
  -c "SELECT id, name, price, stock, status FROM products ORDER BY created_at;"

docker compose exec commerce-db psql -U marketplace -d marketplace_commerce \
  -c "SELECT id, user_id, status, total_amount, discount_amount FROM orders ORDER BY created_at;"

docker compose exec commerce-db psql -U marketplace -d marketplace_commerce \
  -c "SELECT order_id, product_id, quantity, price_at_order FROM order_items ORDER BY order_id;"

docker compose exec commerce-db psql -U marketplace -d marketplace_commerce \
  -c "SELECT code, current_uses, max_uses, active FROM promo_codes ORDER BY code;"
```

Проверка данных User DB без вывода паролей и токенов:

```bash
docker compose exec user-db psql -U marketplace -d marketplace_user \
  -c "SELECT id, email, role, created_at FROM users ORDER BY created_at;"

docker compose exec user-db psql -U marketplace -d marketplace_user \
  -c "SELECT id, user_id, expires_at, created_at FROM refresh_tokens ORDER BY created_at;"
```
