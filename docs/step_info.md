
# 1) Какая это архитектура?

**Слоистая (layered) с “чистой” компоновкой**, близко к **Clean / Hexagonal стилю** без тяжёлого доменного слоя:

```
cmd/
  → app/        (сборка приложения: роутер, мидлвары, сервер)
     → http/    (handlers, middleware)
        → core/ (инфраструктура: конфиг, логи, ошибки, ответы)
           → view (templates/assets — вне Go-кода)
```

### Смысл слоёв:

* **cmd** — точка входа и запуск.
* **app** — композиция: создаёт роутер, вешает middleware, подключает CSRF/статику, готовит http.Server.
* **http** — контроллеры уровня веба: handlers (Home/About/Form/Health/404) и middleware безопасности/кэша.
* **core** — утилиты/инфра: config, logging, унификация ошибок, JSON/RFC7807-ответы.
* **view** — HTML-шаблоны и статика, которыми оперируют handlers.

### 2. Как это работает? (жизненный цикл запроса/процесса)

**Запуск**

1. `cmd/app/main.go`:
   загружает **config**, включает **дневные логи** с автоочисткой, делает sanity-проверки prod, собирает `handler = app.New(...)`, создаёт `srv = app.Server(...)`, слушает порт, на SIGTERM — **graceful shutdown** (10s).

**Сборка приложения**
2. `internal/app/app.go`:
   создаёт `chi.Router` → подключает **middleware**: RequestID, RealIP, Logger, Recoverer, Timeout(15s), **SecureHeaders**, HSTS(если `Secure`), CSRF(gorilla/csrf), кэш статики в prod → регистрирует маршруты (`/`, `/about`, `/form` GET/POST, `/healthz`) и `NotFound`.

3. `internal/app/server.go`:
   отдаёт `http.Server` с «безопасными» таймаутами: ReadHeader 5s / Read 10s / Write 30s / Idle 60s.

**Обработка запроса**
4. Клиент → `chi` роутер → цепочка **middleware**:

* Заголовки безопасности (CSP, XFO, nosniff, COOP, Permissions, Referrer).
* CSRF- защита (токен доступен в шаблонах).
* Таймаут запроса 15s.

5. Роутинг уводит в нужный **handler**:

    * `Home`/`About` рендерят HTML (layout + partials + страница) и вставляют CSRF-поле.
    * `FormIndex` показывает форму; `FormSubmit` валидирует поля (`validator/v10`), при ошибках — ререндер с подсветкой, при успехе — **PRG-редирект** `/form?ok=1`. **БД/email не трогает.**
    * `Health` возвращает `{"status":"ok"}` (200).
    * Неизвестный путь → `NotFound` (404-страница).
6. Если в хендлере случается ошибка для API/JSON-сценариев — через `core/errors.go` + `core/response.go` формируется **RFC7807**-ответ и пишется лог.

**Логи и ротация**
7. `core/logfile.go`:
   вывод идёт сразу в файл **и** в консоль; строки с `ERROR` дублируются в отдельный error-лог; старше 7 дней — удаляются.

**Завершение**
8. На остановку процесса сервер мягко закрывает соединения (`Shutdown` с таймаутом), пишет финальные логи.


## 📁 Структура проекта

``` 

myApp/
├─ cmd/
│  └─ app/
│     └─ main.go                    # entrypoint: конфиг, логи, graceful, запуск app.New()+Server
│
├─ internal/
│  ├─ app/
│  │  ├─ app.go                     # сборка приложения: chi.Router + middleware + статика + маршруты + 404
│  │  └─ server.go                  # http.Server с безопасными таймаутами
│  │
│  ├─ core/
│  │  ├─ config.go                  # конфиг (ENV), проверки для prod
│  │  ├─ errors.go                  # AppError, фабрики (BadRequest, NotFound, Internal…)
│  │  ├─ response.go                # JSON(), NoContent(), Fail() (RFC 7807 style)
│  │  └─ logfile.go                 # логи по датам, авто-ротация и очистка
│  │
│  └─ http/
│     ├─ handler/
│     │  ├─ home.go                 # /
│     │  ├─ about.go                # /about
│     │  ├─ form.go                 # /form (GET/POST) + валидация validator/v10 + PRG
│     │  └─ misc.go                 # /healthz (JSON) и NotFound (404)
│     │
│     └─ middleware/
│        └─ security.go             # CSP, XFO, Referrer, nosniff, Permissions, COOP, HSTS, CacheStatic, Keep-Alive
│
│
├─ web/
│  ├─ assets/                      # статические файлы (CSS/JS/изображения/шрифты)
│  └─ templates/
│     ├─ layouts/base.gohtml
│     ├─ partials/{nav,footer}.gohtml
│     └─ pages/{home,about,form,404}.gohtml
│
├─ logs/                           # генерируется автоматически: DD-MM-YYYY.log (+ errors-DD-MM-YYYY.log, если включено)
├─ go.mod
└─ go.sum

````

# По файлам

## `cmd/app/main.go`

* Загружает конфиг (`core.Load()`), настраивает ежедневные логи с автоочисткой (`core.InitDailyLog()`).
* Prod-проверки: наличие `CSRF_KEY`, предупреждение, если `Secure=false`.
* Сборка `http.Handler` (`app.New`) + запуск `http.Server` (`app.Server`).
* Грациозное завершение по сигналам (`SIGTERM`, `Ctrl+C`), таймаут на shutdown — 10s.
* `derive32()` — SHA-256 из `CSRF_KEY` для CSRF-модуля.

## `internal/app/app.go`

* Создаёт `chi.Router` и цепочку middleware: `RequestID`, `RealIP`, `Logger`, `Recoverer`, `Timeout(15s)`.
* Безопасные заголовки (`mw.SecureHeaders`), keep-alive hint; `HSTS` при `cfg.Secure`.
* Подключает CSRF (`gorilla/csrf`) с `SameSite=Lax`, токен доступен в шаблонах.
* Отдаёт статику `web/assets` (+ долгий кэш в prod).
* Роуты: `/`, `/about`, `/form`(GET/POST), `/healthz`; `NotFound` для 404.

## `internal/app/server.go`

* Возвращает настроенный `*http.Server`:

    * `ReadHeaderTimeout:5s`, `ReadTimeout:10s`, `WriteTimeout:30s`, `IdleTimeout:60s`.
* Таймауты защищают от медленных/висячих соединений.

## `internal/core/config.go`

* Читает ENV: `APP_NAME`, `HTTP_ADDR`, `APP_ENV`, `CSRF_KEY`, `SECURE`.
* Дефолты для dev; в prod валидирует `CSRF_KEY` (завершает на ошибке).

## `internal/core/errors.go`

* `AppError`: единый формат ошибок (код, статус, safe-сообщение, первопричина, поля).
* Фабрики: `BadRequest`, `NotFound`, `Forbidden`, `Unauthorized`, `Internal`.
* `From(err)` приводит любую ошибку к `*AppError` (по умолчанию 500).

## `internal/core/logfile.go`

* Ежедневные логи в `logs/ДД-ММ-ГГГГ.log` и ошибки в `logs/errors-ДД-ММ-ГГГГ.log`.
* Дублирует вывод в файл + консоль; строки с `ERROR` — ещё и в error-лог.
* Асинхронно удаляет логи старше 7 дней.

## `internal/core/response.go`

* `JSON(w,status,v)` — стандартная JSON-ответка.
* `NoContent(w)` — `204`.
* `ProblemDetail` (RFC 7807) и `Fail(w,r,err)` — логирует и отдаёт унифицированную JSON-ошибку.

## `internal/http/handler/home.go`

* Рендерит главную страницу (`base`, `nav`, `footer`, `home`), выставляет `Content-Type`, добавляет CSRF-поле.

## `internal/http/handler/about.go`

* Рендерит страницу «О нас» (`base`, `nav`, `footer`, `about`), с CSRF-полем.

## `internal/http/handler/form.go`

* Показ формы (`FormIndex`): шаблоны, `ok=1` флаг успеха, CSRF-поле.
* Отправка (`FormSubmit`): только `POST`, `ParseForm`, `TrimSpace`, валидация (`validator/v10`), человекочитаемые ошибки.
* При ошибках — ререндер формы с ошибками; при успехе — PRG-редирект на `/form?ok=1`.
* По требованию: **нет** записи в БД и **нет** отправки email.

## `internal/http/handler/common.go`

* `Health` — `200` + `{"status":"ok"}` (простой healthcheck).
* `NotFound` — `404` рендер кастомной страницы.

## `internal/http/middleware/security.go`

* `SecureHeaders` — CSP, XFO=DENY, X-Content-Type-Options=nosniff, Referrer-Policy, Permissions-Policy, COOP.
* `HSTS` — строгий HTTPS (включать при `Secure=true`).
* `CacheStatic` — `Cache-Control: public, max-age=31536000, immutable` для статики (prod).
* `ServerKeepAliveHint` — выставляет `Keep-Alive: timeout=60`.

---

# По всему проекту (архитектурная сводка с привязкой к файлам)

**Точка входа и запуск**

* `cmd/app/main.go`: читает конфиг (`core/config.go`), инициализирует логи (`core/logfile.go`), собирает HTTP-приложение (`internal/app/app.go`), создаёт сервер (`internal/app/server.go`), запускает и корректно гасит по сигналам.

**Конфигурация и инфраструктура**

* `internal/core/config.go`: ENV-конфиг + прод-валидация ключа.
* `internal/core/logfile.go`: дневные логи + автоочистка.
* `internal/core/errors.go` + `internal/core/response.go`: единый формат ошибок (`AppError`) и стандартные JSON-ответы (RFC 7807).

**Веб-приложение**

* `internal/app/app.go`: `chi.Router`, общие middleware (включая безопасность), CSRF, статика, кэш статики (prod), регистрация маршрутов и 404.
* `internal/app/server.go`: безопасные таймауты HTTP.

**Безопасность и производительность**

* `internal/http/middleware/security.go`: CSP/сек-заголовки, HSTS, cache-headers, keep-alive hint.

**Бизнес-маршруты и рендер**

* `internal/http/handler/`:

    * `home.go`, `about.go`: рендер HTML-страниц (layout + partials), CSRF-поле.
    * `form.go`: форма (GET/POST), валидация, UX через PRG, без БД/email.
    * `common.go`: `Health` JSON и `NotFound` 404-страница.

**Статика и шаблоны**

* Отдаются из `/assets` (`internal/app/app.go`), кэшируются в prod.
* Шаблоны: `web/templates/layouts/`, `partials/`, `pages/` (используются во всех обработчиках).


# Что уже хорошо с безопасностью (из коробки)

* **CSRF-защита** (`gorilla/csrf`): токен в формах, `SameSite=Lax`, `HttpOnly`, `Secure` в проде.
* **Жёсткие заголовки** (`middleware.SecureHeaders`):

    * CSP (ограничение источников), `X-Frame-Options=DENY`, `X-Content-Type-Options=nosniff`,
    * `Permissions-Policy`, `Referrer-Policy`, `Cross-Origin-Opener-Policy`.
* **HSTS** в проде (`middleware.HSTS`): принудительный HTTPS на год.
* **Безопасные таймауты** сервера: защита от slowloris/висячих коннектов.
* **Валидация входа** (`validator/v10`) и **PRG-редирект** после POST — меньше повторных отправок.
* **Шаблоны Go** — авто-эскейпинг HTML (XSS-базис закрыт).
* **Логи без PII** по умолчанию (вы сами контролируете, что писать).
* **Санити-чеки продакшена**: запрет дефолтного CSRF-ключа.

# На что обратить внимание / что добавить (по месту в проекте)

1. **Rate limiting & защита от брутфорса**
   Добавить middleware-лимитер (по IP/ключу) перед хендлерами ввода.
   *Где*: `internal/http/middleware` + включить в цепочку в `app/app.go`.

2. **Ограничение размера тела запросов**
   Для форм/API — `http.MaxBytesReader` + проверка `Content-Length`.
   *Где*: в `FormSubmit` и будущих JSON-ручках.

3. **Строже CSP**
   Перейти с общих правил на `nonce`/`sha256` для инлайн-скриптов, убрать `unsafe-inline` для стилей, по возможности.
   *Где*: `middleware.SecureHeaders`.

4. **TLS/прокси и “Secure” режим**
   Убедиться, что `cfg.Secure` привязан к реальному HTTPS (за прокси выставлять `X-Forwarded-Proto` и доверять ТОЛЬКО своему прокси).
   *Где*: `core/config.go` + возможно trust-proxy middleware.

5. **Сессии/куки (если появятся логины)**
   `HttpOnly`, `Secure`, `SameSite=Strict/Lax`, короткие TTL, ключи из ENV.
   *Где*: новый пакет `internal/http/session` или middleware.

6. **CORS (если добавите API для фронта на другом домене)**
   Явно закрыть всё и открыть только нужные origins/методы/заголовки.
   *Где*: middleware CORS в `internal/http/middleware`.

7. **Очистка входных данных**
   Уже есть `TrimSpace`; добавьте нормализацию email/Unicode при необходимости.
   *Где*: в хендлерах и валидаторах.

8. **Скрытие версии/инфры**
   Не отдавать лишние заголовки/баннеры.
   *Где*: сервер/реверс-прокси конфиг + отсутствует в коде (и это хорошо).

9. **Healthcheck доступ**
   `GET /healthz` лучше ограничить по сети/прокси (или отдавать минимум информации).
   *Где*: в ingress/прокси, либо простейший IP-фильтр middleware.

10. **Логи и приватность**
    Следить, чтобы пользовательские поля формы (PII) не попадали в ERROR-логи.
    *Где*: при логировании ошибок/валидации.

11. **Ключи**
    CSRF-ключ — 32 байта криптослучайный (ENV), ротация по регламенту.
    *Где*: `APP_ENV=prod` — уже проверяется; просто используйте надёжные значения.

Итог: архитектура хорошо подходит для “чистого” и безопасного веб-сервиса без лишнего фреймворка. Базовые риски (CSRF, XSS, slowloris, заголовки безопасности) уже прикрыты. Для продакшена добавьте лимитер, ограничение размера тела, строгий CSP с nonce, корректную работу за прокси и, при необходимости, CORS/сессии — это доведёт безопасность до «боевого» уровня.
