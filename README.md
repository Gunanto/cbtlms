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
curl -b cookie.txt -X POST http://localhost:8080/api/v1/auth/logout
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
curl -b cookie.txt -X POST http://localhost:8080/api/v1/admin/registrations/1/approve
```

### Reject registration

```bash
curl -b cookie.txt -X POST http://localhost:8080/api/v1/admin/registrations/1/reject \
  -H 'Content-Type: application/json' \
  -d '{"note":"Data tidak valid"}'
```

## 4) Endpoint attempt (protected, butuh cookie)

### Start attempt

Untuk `siswa/guru`, `student_id` otomatis diambil dari session login.
Untuk `admin/proktor`, `student_id` wajib diisi.

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/attempts/start \
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
  -H 'Content-Type: application/json' \
  -d '{"answer_payload":{"selected":["B"]},"is_doubt":false}'
```

### Submit final

```bash
curl -b cookie.txt -s -X POST http://localhost:8080/api/v1/attempts/1/submit
```

### Get result/review

```bash
curl -b cookie.txt -s http://localhost:8080/api/v1/attempts/1/result
```

## 5) Review policy exam

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

## 6) Ownership rule attempt

- `admin/proktor`: bisa akses semua attempt.
- `siswa/guru`: hanya bisa akses attempt miliknya.
