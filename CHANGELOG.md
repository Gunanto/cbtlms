# Changelog

## 2026-02-19 - commit `31d53d4`

### Added
- Dashboard role-based untuk `admin`, `proktor`, dan `guru` dengan sidebar + panel konten sesuai role.
- Menu `Dashboard` khusus per role sebagai panel terpisah.
- Fitur token ujian/tes untuk proktor.
- Fitur manajemen ujian dan assignment peserta (termasuk enroll by kelas).
- Integrasi Quill.js untuk authoring konten guru (stimuli, naskah soal, opsi jawaban).
- Dukungan tipe soal authoring: `MC`, `MR`, `TF`.
- Import stimuli via CSV + endpoint template CSV.
- Popup detail error import stimuli + unduh laporan gagal CSV.
- Migrasi database:
  - `000012_exam_access_token`
  - `000013_exam_assignments`

### Changed
- UI Admin:
  - `Daftar Pendaftaran` diganti menjadi `Daftar Pengguna`.
  - Data Master dirapikan menjadi tabel terpisah (`Jenjang`, `Sekolah`, `Kelas`) dengan alur CRUD popup.
  - Pengelolaan pengguna diperluas (detail/tambah/ubah/nonaktifkan) dan relasi sekolah/kelas.
- UI Proktor:
  - Sidebar/menu disesuaikan dengan hak akses proktor.
  - Aksi cepat proktor tidak lagi mengarah ke halaman admin.
- UI Guru:
  - Menu guru disusun ulang dan default membuka menu paling atas.
  - Workflow kisi-kisi -> stimuli -> naskah soal diperjelas.
- Import pengguna:
  - Template import + feedback validasi/error diperjelas.
- Security hardening dev infra:
  - Port `Postgres`, `Redis`, dan `MinIO` pada `docker-compose` dibatasi ke `127.0.0.1`.

### Fixed
- Berbagai konflik layout editor (Quill) yang menimpa tombol aksi form.
- Beberapa ketidaksesuaian menu default halaman role saat pertama dibuka.
- Perbaikan akses dan alur CRUD pada halaman manajemen role.

