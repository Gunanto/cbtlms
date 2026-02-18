# Load Test (k6)

Skrip baseline untuk endpoint attempt (`autosave`, `summary`, `submit`).

## Prasyarat
- k6 terpasang.
- Server CBT LMS dan Postgres sudah jalan.
- Punya `attempt_id` + `question_id` valid.
- Punya cookie session dan CSRF token valid.

## Cara ambil cookie + CSRF cepat
Gunakan helper:

```bash
scripts/csrf-curl.sh login admin Admin123!
scripts/csrf-curl.sh token
```

Lalu ambil header cookie (contoh):

```bash
export COOKIE="cbtlms_session=<session-value>; cbtlms_csrf=<csrf-cookie-value>"
export CSRF_TOKEN="<csrf-cookie-value>"
```

## Jalankan

```bash
BASE_URL=http://localhost:8080 \
ATTEMPT_ID=1 \
QUESTION_ID=1 \
COOKIE="$COOKIE" \
CSRF_TOKEN="$CSRF_TOKEN" \
k6 run scripts/loadtest/attempts.js
```

## Catatan
- Ini baseline skeleton, belum mengganti user per VU.
- Untuk uji lebih realistis, buat data banyak siswa + attempt terpisah dan map per VU.
