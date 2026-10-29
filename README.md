# CryptoSignal Lab — Read-only crypto futures screener MVP

An English-language Binance USDT perpetual futures screener with a responsive browser UI, scheduled market scans, RSI analytics, funding rates and optional SMTP alerts. **No trading, custody, user deposits, or live payments.**

## Run with Docker

```bash
cp .env.example .env
# Set ADMIN_API_KEY in .env to a long, random secret.
docker compose up --build -d
```

Visit http://localhost:8080 (bound to localhost by Docker; put behind a TLS reverse proxy if publishing).

## Run with Go

Go 1.23+ is required.

```bash
ADMIN_API_KEY=replace-with-a-long-secret go run ./cmd/server
go test ./...
```

The first scan can take several minutes. The app uses local `screener-data.json` snapshot persistence, which is excluded from Git. `DATABASE_URL` is unused in this MVP; PostgreSQL and Redis are not yet integrated.

## API endpoints

- `GET /api/health`
- `GET /api/markets?q=BTC&maxRsi=40&maxDistance=20&minVolume=500000&limit=100`
- `POST /api/admin/scan` with `Authorization: Bearer YOUR_ADMIN_API_KEY`
- `GET /api/admin/subscriptions` (admin-only)
- `POST /api/admin/subscriptions` (admin-only) body: `{"email":"you@example.com","maxRsi":30,"maxDistance":5,"minVolume":500000}`
- `DELETE /api/admin/subscriptions/{id}` (admin-only)

Subscriptions are **admin-only test records**. SMTP variables enable emails after scans. This is not a public subscription or billing system, and cannot accept paid users yet.

## Market indicators

- Binance USD-M USDT perpetual contracts, volume >= 500,000 USDT
- Four-hour and daily RSI(14) using Wilder smoothing
- Lowest Binance futures daily-candle low across up to 90 completed candles, **not true all-time low**
- Distance to 90-day low, latest reported funding rate, quote-volume filter, heuristic oversold score
- Startup scan then recurring scans with four candle workers
- Existing market snapshot retained when a scan fails

## Before production

- Verify Binance commercial market-data licensing, rate limits, and jurisdictional access.
- Add account authentication, email verification and unsubscribe, queues, per-user scheduling and database persistence.
- Obtain payment-processor approval for crypto analytics and implement server-validated entitlements before charging.
- Harden HTTP service (TLS reverse proxy, security headers, rate limiting), SMTP, logging, backups and data quality checks.
- This repo does **not** offer personal investment advice, signals guaranteeing returns, auto-trading or custody.
