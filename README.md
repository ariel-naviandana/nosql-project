# Banking NoSQL - Digital Banking System

Sistem backend perbankan digital menggunakan **MongoDB** dan **Redis** sebagai implementasi NoSQL.

## Stack Teknologi
- **Golang** (Gin Framework)
- **MongoDB** - Profil nasabah, dokumen KYC, audit log
- **Redis** - Session management, rate limiting
- **Docker & Docker Compose**

## Cara Menjalankan

### Prasyarat
- Docker & Docker Compose terinstall

### 1. Jalankan semua service
```bash
docker-compose up -d
```

### 2. Cek status
```bash
docker-compose ps
docker-compose logs app
```

### 3. Health check
```bash
curl http://localhost:8080/health
```

---

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

### Auth

#### Register Nasabah
```
POST /auth/register
Content-Type: application/json

{
  "nik": "3578012345678901",
  "nama_lengkap": "Budi Santoso",
  "email": "budi@example.com",
  "password": "password123",
  "no_hp": "08123456789",
  "tanggal_lahir": "1990-05-15",
  "alamat": {
    "street": "Jl. Raya Darmo No. 10",
    "city": "Surabaya",
    "province": "Jawa Timur",
    "postal_code": "60241"
  }
}
```

#### Login
```
POST /auth/login
Content-Type: application/json

{
  "email": "budi@example.com",
  "password": "password123"
}
```
Response: `{ "token": "uuid-token", ... }` — gunakan token ini di header selanjutnya.

#### Logout
```
POST /auth/logout
Authorization: Bearer <token>
```

---

### Nasabah (Authenticated)

#### Get Profil
```
GET /nasabah/profile
Authorization: Bearer <token>
```

#### Update Profil
```
PUT /nasabah/profile
Authorization: Bearer <token>

{
  "no_hp": "08987654321",
  "alamat": {
    "street": "Jl. Pemuda No. 5",
    "city": "Surabaya",
    "province": "Jawa Timur",
    "postal_code": "60271"
  }
}
```

#### Upload Dokumen KYC
```
POST /nasabah/kyc
Authorization: Bearer <token>

{
  "tipe_dokumen": "ktp",
  "nama_file": "ktp_budi.jpg",
  "mime_type": "image/jpeg",
  "file_base64": "<base64_string>"
}
```
Tipe dokumen: `ktp`, `selfie`, `npwp`, `signature`

#### Lihat Dokumen KYC Saya
```
GET /nasabah/kyc
Authorization: Bearer <token>
```

#### Lihat Audit Log Saya
```
GET /nasabah/audit-log?page=1&limit=20
Authorization: Bearer <token>
```

#### Info Session & Rate Limit (Demo Redis)
```
GET /nasabah/session-info
Authorization: Bearer <token>
```

---

### Admin

#### Lihat Semua Nasabah
```
GET /admin/nasabah?page=1&limit=10
Authorization: Bearer <token>
```

#### Verifikasi KYC
```
PUT /admin/kyc/:id/verify
Authorization: Bearer <token>

{
  "status": "verified",
  "keterangan": "Dokumen valid"
}
```

#### Lihat Semua Audit Log
```
GET /admin/audit-log?page=1&limit=20
Authorization: Bearer <token>
```

---

## Arsitektur Database

### MongoDB Collections
| Collection | Kegunaan |
|---|---|
| `nasabah` | Profil lengkap nasabah, data akun |
| `dokumen_kyc` | Dokumen KYC (KTP, selfie, NPWP) |
| `audit_log` | Log semua aktivitas sistem |

### Redis Keys
| Pattern | Kegunaan | TTL |
|---|---|---|
| `session:<token>` | Session login nasabah | 24 jam |
| `ratelimit:login:<ip>` | Counter percobaan login per IP | 15 menit |
| `ratelimit:api:<ip>` | Counter request API per IP | 1 menit |
| `blacklist:<token>` | Token yang sudah logout | 24 jam |
| `saldo:<account_id>` | Cache saldo (ref PostgreSQL) | Configurable |

---

## Menghentikan Service
```bash
docker-compose down
# Hapus data juga:
docker-compose down -v
```
