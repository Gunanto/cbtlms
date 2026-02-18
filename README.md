# CBT LMS Starter

Starter project CBT LMS berbasis Go (SSR), PostgreSQL, Redis, dan MinIO.

## Jalankan dependency lokal

```bash
docker compose -f deployments/docker-compose.yml up -d
```

## Jalankan aplikasi

```bash
go mod tidy
go run ./cmd/web
```

Aplikasi aktif di `http://localhost:8080`.

Tuning DB pool via environment (opsional):
- `DB_MAX_OPEN_CONNS` (default `25`)
- `DB_MAX_IDLE_CONNS` (default `25`)
- `DB_CONN_MAX_LIFETIME_MINUTES` (default `30`)

## Health check

```bash
curl -s http://localhost:8080/healthz
```

## Migration SQL

Urutan migration saat ini:
- `000001_init_schema`
- `000002_authoring_review_features`
- `000003_audit_triggers_and_seed`
- `000004_auth_models`
- `000005_auth_sessions`
- `000006_auth_security_and_seed`
- `000007_review_policy_constraint`
- `000008_master_data_school_class_enrollment`
- `000009_attempt_events_anticheat`
- `000010_performance_indexes`

Gunakan runner:

```bash
./scripts/migrate.sh status
./scripts/migrate.sh up
./scripts/migrate.sh down 1
```

## Catatan admin/proktor

- Untuk MVP, admin dan proktor memakai dashboard operasional yang sama.
- Pembeda utama ada di permission detail yang bisa ditambah bertahap.

## Seed akun awal (migration 000006)

- `admin / Admin123!`
- `proktor / Proktor123!`

Segera ganti password awal via endpoint bootstrap.

## Daftar endpoint + cURL (yang sudah berjalan)

## Format response API (Step 8)

Semua endpoint API utama sekarang memakai envelope konsisten:

```json
{
  "ok": true,
  "data": {},
  "error": null,
  "meta": {
    "request_id": "req-id-dari-middleware"
  }
}
```

Contoh saat error:

```json
{
  "ok": false,
  "error": {
    "code": "invalid_request",
    "message": "invalid request body"
  },
  "meta": {
    "request_id": "req-id-dari-middleware"
  }
}
```

Kode error utama:
- `invalid_request` (400)
- `unauthorized` (401)
- `forbidden` (403)
- `not_found` (404)
- `conflict` (409)
- `unprocessable_entity` (422)
- `rate_limited` (429)
- `internal_error` (500)

## 0) Bootstrap akun awal

Perlu set env `BOOTSTRAP_TOKEN`.

```bash
curl -s -X POST http://localhost:8080/api/v1/bootstrap/init \
  -H 'Content-Type: application/json' \
  -d '{
    "token":"CHANGE_THIS_BOOTSTRAP_TOKEN",
    "admin_username":"admin",
    "admin_email":"admin@school.local",
    "admin_password":"AdminStrong123!",
    "proktor_username":"proktor",
    "proktor_email":"proktor@school.local",
    "proktor_password":"ProktorStrong123!"
  }'
```

## 1) Auth

### Login password

```bash
curl -i -c cookie.txt -X POST http://localhost:8080/api/v1/auth/login-password \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"admin","password":"Admin123!"}'
```

Ambil CSRF token dari cookie setelah login:

```bash
export CSRF_TOKEN="$(awk '$6==\"cbtlms_csrf\"{print $7}' cookie.txt | tail -n1)"
echo "$CSRF_TOKEN"
```

### Request OTP

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/otp/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@school.local"}'
```

Catatan:
- Jika SMTP aktif, OTP dikirim via email.
- Jika SMTP belum aktif, OTP tampil di log server (`[DEV-OTP]`).

### Verify OTP

```bash
curl -i -c cookie.txt -X POST http://localhost:8080/api/v1/auth/otp/verify \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@school.local","code":"123456"}'
```

### Me

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/auth/me
```

### Logout

```bash
curl -b cookie.txt -X POST http://localhost:8080/api/v1/auth/logout \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

## 2) Registrasi mandiri

### Buat pendaftaran

```bash
curl -s -X POST http://localhost:8080/api/v1/registrations \
  -H 'Content-Type: application/json' \
  -d '{
    "role_requested":"siswa",
    "email":"siswa1@example.com",
    "password":"Siswa123!",
    "full_name":"Siswa Satu",
    "phone":"081234567890",
    "institution_name":"SMA Demo",
    "form_payload":{"nisn":"0011223344","class_name":"X-1"}
  }'
```

## 3) Endpoint admin/proktor

### List pending registrations

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/admin/registrations?limit=50'
```

### Approve registration

```bash
curl -b cookie.txt -X POST http://localhost:8080/api/v1/admin/registrations/1/approve \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

### Reject registration

```bash
curl -b cookie.txt -X POST http://localhost:8080/api/v1/admin/registrations/1/reject \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"note":"Data tidak valid"}'
```

## 4) Master data sekolah/kelas/peserta (Step 2)

### Create school

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/admin/schools \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"SMA Negeri Contoh",
    "code":"SMAN-CONT",
    "address":"Jl. Pendidikan No. 1"
  }'
```

### Create class

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/admin/classes \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "school_id":1,
    "name":"X-1",
    "grade_level":"10"
  }'
```

### Import students CSV

Contoh file `students.csv`:

```csv
full_name,username,password,email,nisn,school_name,class_name,grade_level
Siswa Satu,siswa_satu,Password123!,siswa1@example.com,1234567890,SMA Negeri Contoh,X-1,10
Siswa Dua,siswa_dua,Password123!,siswa2@example.com,2234567890,SMA Negeri Contoh,X-1,10
```

Import:

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/admin/imports/students \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -F "file=@students.csv"
```

## 5) Endpoint attempt (protected, butuh cookie)

### Start attempt

Untuk `siswa/guru`, `student_id` otomatis diambil dari session login.
Untuk `admin/proktor`, `student_id` wajib diisi.

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/attempts/start \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"exam_id":1,"student_id":1001}'
```

### Get attempt summary

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/attempts/1
```

### Save answer

```bash
curl -b cookie.txt -s -X PUT http://localhost:8080/api/v1/attempts/1/answers/10 \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"answer_payload":{"selected":["B"]},"is_doubt":false}'
```

### Submit final

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/attempts/1/submit \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

### Get result/review

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/attempts/1/result
```

## 6) Scoring contract (Step 4)

Standar format `answer_key` (di `question_versions.answer_key`) dan `answer_payload` (di `attempt_answers.answer_payload`):

- `pg_tunggal`
  - `answer_key`: `{"correct":"B"}`
  - `answer_payload`: `{"selected":"B"}` atau `{"selected":["B"]}`
- `multi_jawaban` (sementara policy `exact`)
  - `answer_key`: `{"correct":["A","D"],"mode":"exact"}`
  - `answer_payload`: `{"selected":["A","D"]}`
- `benar_salah_pernyataan`
  - `answer_key`: `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`
  - `answer_payload`: `{"answers":[{"id":"s1","value":true},{"id":"s2","value":false}]}`

Aturan scoring:
- `pg_tunggal`: benar penuh = `weight`, selain itu `0`.
- `multi_jawaban`: harus exact match (set sama persis), benar penuh = `weight`, selain itu `0`.
- `benar_salah_pernyataan`: skor per pernyataan = `weight / jumlah_pernyataan`, dijumlahkan.
- `unanswered`: `is_correct = null`, `earned_score = 0`.
- `malformed_payload`: dianggap terjawab tapi salah, `is_correct = false`, `earned_score = 0`.
- `malformed_answer_key`: aman (tidak panic), `earned_score = 0` dan perlu perbaikan data soal.

Feedback scoring disimpan di `attempt_scores.feedback`:
- `selected`
- `correct`
- `reason` (`correct`, `wrong`, `partial`, `unanswered`, `malformed_payload`, `malformed_answer_key`)
- `breakdown` (khusus `benar_salah_pernyataan`)

## 7) Review policy exam

Nilai `exams.review_policy` yang didukung:
- `after_submit` (default): hasil bisa dilihat setelah attempt final (`submitted`/`expired`).
- `after_exam_end`: hasil baru bisa dilihat setelah `exams.end_at` terlewati.
- `immediate`: hasil bisa dilihat segera setelah attempt final.
- `disabled`: hasil/review selalu ditutup.

Contoh update policy:

```bash
docker exec -i cbtlms-postgres psql -U cbtlms -d cbtlms -c \
  \"UPDATE exams SET review_policy='after_exam_end', end_at = now() + interval '1 hour' WHERE id=1;\"
```

## 8) Ownership rule attempt

- `admin/proktor`: bisa akses semua attempt.
- `siswa/guru`: hanya bisa akses attempt miliknya.

## 9) Anti-cheat event log (Step 7)

Jenis event yang didukung:
- `tab_blur`
- `reconnect`
- `rapid_refresh`
- `fullscreen_exit`

### Kirim event dari client (owner/admin/proktor)

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/attempts/1/events \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "event_type":"tab_blur",
    "client_ts":"2026-02-18T19:30:00+07:00",
    "payload":{"tab_hidden_ms":1200}
  }'
```

### Lihat event attempt (admin/proktor)

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/attempts/1/events?limit=200'
```

## 10) Stimulus API (authoring)

Role yang diizinkan: `admin`, `proktor`, `guru`.

### Create stimulus

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/stimuli \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "subject_id": 1,
    "title": "Stimulus Pecahan",
    "stimulus_type": "single",
    "content": {
      "body": "<p>Bacaan/stimulus di sini</p>"
    }
  }'
```

Contoh `multiteks`:

```json
{
  "subject_id": 1,
  "title": "Stimulus Multi Teks",
  "stimulus_type": "multiteks",
  "content": {
    "tabs": [
      {"title":"Teks 1","body":"Isi teks pertama"},
      {"title":"Teks 2","body":"Isi teks kedua"}
    ]
  }
}
```

### List stimuli by subject

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/stimuli?subject_id=1'
```

## 11) Question versioning API

Role yang diizinkan: `admin`, `proktor`, `guru`.

### Create new version

Endpoint ini selalu membuat versi baru (`version_no` naik), tidak overwrite versi lama.

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/questions/10/versions \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "stem_html":"<p>Soal versi terbaru</p>",
    "explanation_html":"<p>Pembahasan versi terbaru</p>",
    "hint_html":"<p>Hint singkat</p>",
    "answer_key":{"correct":"B"},
    "duration_seconds":60,
    "weight":1.5,
    "change_note":"Perbaikan redaksi"
  }'
```

### Finalize version

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/questions/10/versions/2/finalize \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

### List versions

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/questions/10/versions
```

## 12) Question parallels API (per exam)

Role yang diizinkan: `admin`, `proktor`, `guru`.

### Create parallel mapping

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/exams/3/parallels \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "question_id": 10,
    "parallel_group": "default",
    "parallel_order": 1,
    "parallel_label": "pararel_1"
  }'
```

### List parallel mapping

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/exams/3/parallels?parallel_group=default'
```

### Update parallel mapping

```bash
curl -b cookie.txt -s -X PUT http://localhost:8080/api/v1/exams/3/parallels/1 \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "question_id": 12,
    "parallel_group": "default",
    "parallel_order": 2,
    "parallel_label": "pararel_2"
  }'
```

### Delete parallel mapping

```bash
curl -b cookie.txt -s -X DELETE http://localhost:8080/api/v1/exams/3/parallels/1 \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

## 13) Review / Telaah API (Step 6)

Role yang diizinkan: `admin`, `proktor`, `guru`.

### Assign review task

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/reviews/tasks \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "question_version_id": 12,
    "exam_id": 3,
    "reviewer_id": 77,
    "note": "Mohon telaah kualitas soal"
  }'
```

### Decision review task

`status` yang didukung:
- `disetujui`
- `perlu_revisi` (wajib isi `note`)

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/reviews/tasks/10/decision \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "status":"perlu_revisi",
    "note":"Distraktor opsi C masih ambigu."
  }'
```

### List review task (reviewer only by default)

Untuk role `guru`, `reviewer_id` otomatis dipaksa ke user login.
Untuk `admin/proktor`, bisa filter reviewer lain.

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/reviews/tasks?status=menunggu_reviu'
```

```bash
curl -b cookie.txt -s 'http://localhost:8080/api/v1/reviews/tasks?status=menunggu_reviu&reviewer_id=77'
```

### Get review history per question

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/questions/10/reviews
```

## 13) Observability (Step 9)

### Structured request log

Server sekarang mencatat log JSON per request dengan field utama:
- `request_id`
- `user_id`
- `attempt_id`
- `method`
- `path`
- `status`
- `latency_ms`
- `remote_ip`

### Metrics endpoint

```bash
curl -s http://localhost:8080/metrics
```

Metrik utama yang tersedia:
- `cbtlms_http_requests_total{method,path,status}`
- `cbtlms_http_request_latency_ms_sum{...}`
- `cbtlms_http_request_latency_ms_avg{...}`
- `cbtlms_db_open_connections`
- `cbtlms_db_in_use_connections`
- `cbtlms_db_idle_connections`
- `cbtlms_db_wait_count`
- `cbtlms_db_wait_duration_ms`

Catatan:
- Metrik `autosave` dan `submit` sudah tercakup via label path:
  - `/api/v1/attempts/{id}/answers/{id}` untuk autosave
  - `/api/v1/attempts/{id}/submit` untuk submit

## 14) Performa & Query Hardening (Step 10)

Index tambahan untuk endpoint kritikal ditambahkan di migration:
- `000010_performance_indexes`

Verifikasi query plan (contoh):

```bash
docker exec -i cbtlms-postgres psql -U cbtlms -d cbtlms -c \
  "EXPLAIN ANALYZE SELECT option_key FROM question_options WHERE question_id=1 AND is_correct=TRUE;"
```

```bash
docker exec -i cbtlms-postgres psql -U cbtlms -d cbtlms -c \
  "EXPLAIN ANALYZE SELECT COUNT(*) FROM attempt_answers WHERE attempt_id=1 AND is_doubt=TRUE;"
```

```bash
docker exec -i cbtlms-postgres psql -U cbtlms -d cbtlms -c \
  "EXPLAIN ANALYZE SELECT id FROM attempts WHERE student_id=1001 AND status='in_progress';"
```

## 15) Security Hardening (Step 12)

### Rate limit endpoint sensitif

Public auth endpoint dilindungi rate-limit berbasis IP+path.

Env:
- `AUTH_RATE_LIMIT_PER_MINUTE` (default `60`)

### CSRF protection (session cookie)

Server menerbitkan CSRF token saat login/OTP verify:
- Cookie: `cbtlms_csrf`
- Header response: `X-CSRF-Token`

Jika `CSRF_ENFORCED=true`, setiap request mutasi pada route terautentikasi wajib kirim:
- Header `X-CSRF-Token` dengan nilai yang sama dengan cookie `cbtlms_csrf`.

Env:
- `CSRF_ENFORCED=true` (default sekarang)
- set `false` hanya untuk development tertentu jika benar-benar diperlukan.

### Runbook security & backup

Lihat:
- `docs/SECURITY_RUNBOOK.md`

### Helper script untuk curl + CSRF

Tersedia script:
- `scripts/csrf-curl.sh`

Contoh:

```bash
scripts/csrf-curl.sh login admin Admin123!
scripts/csrf-curl.sh req GET /api/v1/auth/me
scripts/csrf-curl.sh req POST /api/v1/auth/logout
```

## 16) Load Test Baseline (Step 11)

Skeleton load test tersedia:
- `scripts/loadtest/attempts.js`
- `scripts/loadtest/README.md`

Contoh menjalankan:

```bash
BASE_URL=http://localhost:8080 \
ATTEMPT_ID=1 \
QUESTION_ID=1 \
COOKIE="$COOKIE" \
CSRF_TOKEN="$CSRF_TOKEN" \
k6 run scripts/loadtest/attempts.js
```

## 17) CI Workflow

Workflow GitHub Actions tersedia di:
- `.github/workflows/ci.yml`

Job yang ada:
- `unit-test` (jalan pada push/PR)
- `integration-test` Postgres (jalan saat `workflow_dispatch`)

## 18) Backup & Restore Check Script

Script:
- `scripts/backup.sh`
- `scripts/restore_check.sh`

Backup:

```bash
scripts/backup.sh
```

Contoh override output:

```bash
BACKUP_DIR=./tmp/backups COMPRESS=true RETENTION_DAYS=14 scripts/backup.sh
```

Restore check (non-destruktif, pakai temp DB):

```bash
scripts/restore_check.sh /tmp/cbtlms-backups/cbtlms_cbtlms_YYYYMMDD_HHMMSS.sql.gz
```
