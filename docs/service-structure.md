```shell

gateway/
├─ cmd/
│  └─ gateway/
│     └─ main.go                   # точка входа: конфиг, DI, запуск WS/HTTP
│
├─ internal/
│  ├─ domain/                      # бизнес-модель, чистые сущности/ценности
│  │  ├─ entities/
│  │  │  ├─ user.go
│  │  │  ├─ dialog.go
│  │  │  └─ message.go
│  │  ├─ valueobjects/
│  │  │  ├─ ids.go                 # UserID, DialogID, MessageID, ClientMessageID
│  │  │  └─ body.go                # текст/аттачи + валидация размеров
│  │  └─ errors.go
│  │
│  ├─ application/                 # use-case слой + порты (интерфейсы)
│  │  ├─ ports/
│  │  │  ├─ inbound.go             # интерфейсы входа (Subscribe/Send/Typing/Read)
│  │  │  └─ outbound.go            # StoreClient, EventBus, Cache, Auth, Clock
│  │  ├─ usecase/
│  │  │  ├─ send_message.go
│  │  │  ├─ subscribe.go
│  │  │  ├─ typing.go
│  │  │  └─ read_upto.go
│  │  ├─ dto/
│  │  │  ├─ send_message_cmd.go
│  │  │  ├─ subscribe_cmd.go
│  │  │  └─ events.go              # message_new/receipt/presence (без транспорта)
│  │  └─ services/
│  │     ├─ validator.go
│  │     └─ ratelimiter.go
│  │
│  ├─ adapters/                    # реализации портов (вход/выход)
│  │  ├─ inbound/
│  │  │  ├─ ws/
│  │  │  │  ├─ server.go           # апгрейд, handshake (JWT), ping/pong
│  │  │  │  ├─ conn.go             # модель соединения, хаб, подписки
│  │  │  │  ├─ router.go           # разбор JSON {type:...} -> handler
│  │  │  │  ├─ handlers.go         # send_message/typing/read/subscribe
│  │  │  │  ├─ messages_map.go     # маппинг WS<->application DTO
│  │  │  │  └─ middleware.go       # пер-ю аутентификация, rate limit
│  │  │  └─ http/
│  │  │     ├─ healthz.go          # /healthz
│  │  │     ├─ metrics.go          # /metrics (Prometheus)
│  │  │     └─ pprof.go            # /debug/pprof (опционально)
│  │  └─ outbound/
│  │     ├─ storeclient/
│  │     │  ├─ grpc_client.go      # /internal/messages.Send, ReadUpto, CheckMembership
│  │     │  └─ mapper.go
│  │     ├─ eventbus/
│  │     │  ├─ redis_pubsub.go     # подписка на события Store (fan-out)
│  │     │  └─ streams.go          # Redis Streams для resume (last_event_id)
│  │     ├─ cache/
│  │     │  └─ redis_cache.go      # presence/typing/idempotency
│  │     ├─ auth/
│  │     │  └─ jwks_verifier.go    # проверка JWT (kid, подпись, exp, scope)
│  │     └─ observability/
│  │        ├─ logger.go
│  │        ├─ metrics.go
│  │        └─ tracing.go
│  │
│  ├─ app/                         # композиция/DI и lifecycle
│  │  ├─ container.go              # сборка зависимостей
│  │  └─ lifecycle.go              # старт/стоп серверов, graceful shutdown
│  │
│  └─ config/
│     └─ types.go                  # структуры конфигурации
│
├─ api/
│  └─ proto/
│     ├─ message_store.proto       # контракты внутреннего gRPC (вендор или git submodule)
│     └─ common.proto
│
├─ gen/
│  └─ pb/                          # сгенерированные gRPC-клиенты (go)
│
├─ pkg/                            # утилиты без зависимостей на internal
│  ├─ backoff/
│  ├─ xcontext/
│  ├─ xerrors/
│  └─ xws/                         # вспомогалки для WS (frame limits, write pump)
│
├─ configs/
│  └─ config.yaml                  # значения по умолчанию
│
├─ deployments/
│  ├─ docker/
│  │  └─ Dockerfile
│  └─ k8s/
│     ├─ deployment.yaml
│     ├─ service.yaml
│     ├─ hpa.yaml
│     └─ configmap.yaml
│
├─ test/
│  ├─ e2e/
│  │  └─ send_and_read_test.go     # «сквозняк» WS↔️Store (с testcontainers)
│  └─ integration/
│     ├─ ws_resume_test.go
│     └─ idempotency_test.go
│
├─ docs/
│  ├─ ws-contract.md               # JSON-протокол клиента (типы событий)
│  └─ architecture.md
│
├─ scripts/
│  ├─ gen.sh                       # генерация gRPC
│  └─ run_local.sh                 # локальный запуск (redis+store mock)
│
├─ .env.example
├─ .golangci.yml
├─ .gitignore
├─ go.mod
├─ go.sum
├─ Makefile
└─ README.md
```
