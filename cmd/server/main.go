package main

import (
 "cryptoscreener/internal/app"
 "log"
 "os"
)

func main() {
 cfg := app.Config{Addr: env("ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), BinanceBase: env("BINANCE_BASE_URL", "https://fapi.binance.com"), ScanInterval: env("SCAN_INTERVAL", "4h"), SMTPHost: os.Getenv("SMTP_HOST"), SMTPPort: env("SMTP_PORT", "587"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPass: os.Getenv("SMTP_PASS"), SMTPFrom: os.Getenv("SMTP_FROM"), APIKey: os.Getenv("ADMIN_API_KEY")}
 if err := app.Run(cfg); err != nil { log.Fatal(err) }
}
func env(k, d string) string {
 if v := os.Getenv(k); v != "" { return v }
 return d
}
