backend/go/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── worker/
│       └── main.go
│
├── api/
│   ├── routes.go
│   ├── middleware.go
│   ├── handlers_auth.go
│   ├── handlers_user.go
│   ├── handlers_posts.go
│   ├── handlers_chat.go
│   ├── handlers_tournaments.go
│   ├── handlers_notifications.go
│   └── handlers_health.go
│
├── auth/
│   ├── jwt.go
│   ├── tokens.go
│   └── passwd_helpers.go
│
├── config/
│   └── config.go
│
├── db/
│   ├── conn.go
│   ├── migrator.go
│   └── migrations/
│       ├── 0001_gatherup_schema_with_softdelete.sql
│       └── 0002_seed_lookup_data.sql
│
├── models/
│   ├── user.go
│   ├── post.go
│   ├── message.go
│   ├── tournament.go
│   ├── notification.go
│   └── audit.go
│
├── repository/
│   ├── user_repo.go
│   ├── post_repo.go
│   ├── message_repo.go
│   ├── tournament_repo.go
│   └── notification_repo.go
│
├── service/
│   ├── auth_service.go
│   ├── user_service.go
│   ├── post_service.go
│   ├── chat_service.go
│   ├── tournament_service.go
│   └── notification_service.go
│
├── ws/
│   ├── hub.go
│   ├── connection.go
│   ├── router.go
│   ├── handlers.go
│   └── redis_pubsub.go
│
├── worker/
│   ├── jobs.go
│   ├── notifier.go
│   ├── leaderboard.go
│   └── cleanup.go
│
├── storage/
│   ├── r2_client.go
│   ├── upload_handler.go
│   └── file_utils.go
│
├── cache/
│   ├── redis.go
│   ├── user_cache.go
│   └── feed_cache.go
│
├── shared/
│   ├── types.go
│   ├── constants.go
│   └── utils.go
│
├── internal/
│   ├── errors.go
│   ├── validators.go
│   └── serializers.go
│
├── pkg/
│   ├── logger/
│   │   └── logger.go
│   └── metrics/
│       └── metrics.go
│
├── taskqueue/
│   ├── taskqueue.go
│   ├── redis_queue.go
│   └── tasks.go
│
├── search/
│   ├── elasticsearch.go
│   └── post_search.go
│
├── proto/
│   ├── api.proto
│   └── message.proto
│
├── tools/
│   ├── dev-compose.yml
│   └── load-test/
│       └── websocket_test.go
│
├── scripts/
│   ├── migrate.sh
│   ├── seed_sample_data.sh
│   └── healthcheck.sh
│
├── infra/
│   ├── docker/
│   │   ├── Dockerfile.api
│   │   ├── Dockerfile.worker
│   │   └── docker-compose.prod.yml
│   └── k8s/
│       ├── api-deployment.yaml
│       ├── ws-deployment.yaml
│       ├── worker-deployment.yaml
│       └── service.yaml
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── security-scan.yml
│
├── go.mod
├── go.sum
├── Makefile
├── .env.example
└── README.md
