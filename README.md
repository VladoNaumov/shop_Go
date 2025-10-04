

```
shop/
├─ cmd/
│  └─ app/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ platform/
│  │  └─ server.go
│  └─ transport/
│     └─ httpx/
│        ├─ router.go
│        ├─ middleware.go
│        └─ handlers/
│           ├─ health.go
│           └─ home.go
├─ web/
│  ├─ assets/
│  │  └─ css/
│  │     └─ style.css
│  └─ templates/
│     ├─ layouts/
│     │  └─ base.tmpl
│     ├─ partials/
│     │  ├─ nav.tmpl
│     │  └─ footer.tmpl
│     └─ pages/
│        └─ home.tmpl
├─ .env.example
├─ go.mod
└─ Makefile

```

- cmd/app — точка входа.
- internal/ — ваш код (инкапсулирован от внешнего импорта).
- transport/httpx — HTTP-слой (роутер, middleware, хендлеры).
- platform — инфраструктурные штуки (HTTP-сервер, таймауты).
- web/ — шаблоны и статика.