# Kafka Event-Driven Activity Log

Implementasi event-driven architecture menggunakan Kafka untuk audit trail activity user.

## Arsitektur

```
HTTP POST /usernotes
  └─> usernotes.SaveNote()
       └─> Postgres TX (atomic):
            ├─ INSERT note
            └─ INSERT outbox (note.created event)   <-- dual-write terselesaikan

OutboxRelay (background, poll tiap 1s)
  └─> SELECT outbox WHERE processed_at IS NULL
       ├─> Kafka Producer: publish event
       └─> UPDATE outbox SET processed_at (atau retries++ kalau gagal)

Kafka Consumer (background goroutine)
  └─> consume note.created event
       └─> Postgres: INSERT activity_log

HTTP GET /activity
  └─> Query activity_log (sourced dari Kafka consumer)
```

Full pipeline: **note → outbox (atomic) → relay → Kafka → consumer → activity_log → API**.

## Setup

1. **Tambah KAFKA_BROKERS di .envrc**:
   ```bash
   export KAFKA_BROKERS=kafka:9092
   ```

2. **Start services**:
   ```bash
   cd docker
   docker-compose up -d
   ```

3. **Run migrations** (jika belum):
   ```bash
   make migrate-up  # atau manual: psql < db/migrations/000003_activity_log.up.sql
   ```

## Test End-to-End

1. Register & login untuk dapat token
2. Create note (trigger event):
   ```bash
   curl -X POST http://localhost:8080/usernotes \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"title":"Test","content":"Hello Kafka"}'
   ```

3. Query activity log (beberapa detik kemudian, setelah consumer proses):
   ```bash
   curl http://localhost:8080/activity?limit=10 \
     -H "Authorization: Bearer $TOKEN"
   ```

   Response:
   ```json
   {
     "data": [
       {
         "id": "uuid",
         "event_type": "note.created",
         "user_id": "user-uuid",
         "entity_type": "note",
         "entity_id": "note-uuid",
         "metadata": {},
         "created_at": "2026-08-05T..."
       }
     ]
   }
   ```

## Graceful Degradation

Kafka **optional**: jika `KAFKA_BROKERS` kosong atau Kafka down, app tetap jalan normal. Tanpa Kafka, save hook nil → note tersimpan tanpa outbox entry, endpoint `/activity` return empty array.

## Transactional Outbox (Phase 2 — implemented)

Masalah **dual-write**: kalau `INSERT note` sukses tapi publish Kafka gagal (broker down), event hilang padahal note tersimpan → data inconsistent.

Solusi outbox:

1. **Atomic write** — `note.created` ditulis ke tabel `outbox` di **transaksi yang sama** dengan `INSERT note` (`internal/usernotes/store_postgres.go` + `SaveHook`). Kedua-duanya commit atau kedua-duanya rollback.
2. **Relay** (`internal/events/outbox_relay.go`) — poll `outbox WHERE processed_at IS NULL` tiap 1 detik, publish ke Kafka, lalu `MarkProcessed`. Kalau Kafka down, event tetap di DB dan di-retry (kolom `retries`).
3. **At-least-once + idempotency** — consumer commit-after-handle, jadi event dijamin sampai minimal 1x. Duplikasi (jika ada) di-handle di sisi consumer.

**Trade-off yang disengaja** (`ponytail`):
- Relay pakai polling 1s, bukan CDC/Debezium. Lebih simpel, latency ~1s. Upgrade ke Debezium kalau butuh sub-second / high-throughput.
- `GetPending` limit 100 per batch, tanpa `FOR UPDATE SKIP LOCKED`. Aman untuk single relay instance. Kalau mau multi-instance relay, tambah row locking biar gak double-publish.

## Next (opsional, kalau mau lanjut)

- **Dead-letter**: event yang gagal N kali (lihat kolom `retries`) dipindah ke topic DLQ.
- **Multi consumer group**: fan-out satu event ke analytics + search indexer.
