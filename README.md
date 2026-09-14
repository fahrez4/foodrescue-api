# FoodRescue REST API

REST API backend untuk aplikasi mobile Flutter **FoodRescue** menggunakan bahasa **Go (Golang)** dan **Gin Framework**, lengkap dengan operasi **CRUD** penuh dan dokumentasi interaktif **Swagger UI**.

---

## 🛠️ Tech Stack & Dependencies

- **Language & Framework**: Go (`1.27+`), Gin Framework (`github.com/gin-gonic/gin`)
- **Documentation**: Swagger UI (`github.com/swaggo/gin-swagger`, `github.com/swaggo/files`, `github.com/swaggo/swag`)
- **Database**: MySQL Remote Hostinger (`github.com/go-sql-driver/mysql`)
- **Authentication**: JWT Bearer Token (`github.com/golang-jwt/jwt/v5`)
- **Identifier**: UUID v4 (`github.com/google/uuid`)
- **Password Security**: Bcrypt (`golang.org/x/crypto/bcrypt`)

---

## 📁 Struktur Direktori

```text
foodrescue-api/
├── main.go
├── config/
│   └── database.go
├── controllers/
│   ├── auth_controller.go
│   ├── food_controller.go
│   └── claim_controller.go
├── docs/
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── middlewares/
│   └── auth_middleware.go
├── models/
│   └── response.go
├── go.mod
├── go.sum
└── .gitignore
```

---

## 📖 Dokumentasi Interaktif Swagger UI

Setelah server dijalankan, buka browser dan akses URL berikut:
👉 **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

Untuk endpoint terproteksi (`BearerAuth`), klik tombol **Authorize** di pojok kanan atas Swagger UI, lalu masukkan:
```text
Bearer <token_jwt_hasil_login>
```

---

## 🚀 Menjalankan Aplikasi

1. **Unduh dependencies**:
   ```bash
   go mod tidy
   ```

2. **Generate / Re-generate Swagger**:
   ```bash
   swag init
   ```

3. **Jalankan server**:
   ```bash
   go run main.go
   ```
   Server berjalan di port `:8080` (`http://localhost:8080`).

---

## 📡 Matriks Endpoint API & Status CRUD

| Operasi CRUD | Method | Endpoint | Akses | Keterangan |
|---|---|---|---|---|
| **CREATE (User)** | `POST` | `/api/v1/register` | Public | Registrasi pengguna baru |
| **READ (Auth)** | `POST` | `/api/v1/login` | Public | Login & peroleh JWT 7 hari |
| **READ (All)** | `GET` | `/api/v1/foods` | Public | Daftar surplus makanan aktif |
| **READ (Detail)** | `GET` | `/api/v1/foods/:id` | Public | Detail makanan berdasarkan ID |
| **READ (My Foods)** | `GET` | `/api/v1/my-foods` | Protected | Riwayat makanan donasi milik saya |
| **CREATE (Food)** | `POST` | `/api/v1/foods` | Protected | Posting donasi surplus makanan baru |
| **UPDATE (Food)** | `PUT` | `/api/v1/foods/:id` | Protected | Edit donasi makanan (hanya pemilik & status `available`) |
| **DELETE (Food)** | `DELETE` | `/api/v1/foods/:id` | Protected | Hapus postingan donasi (hanya pemilik & status `available`) |
| **ACTION (Claim)** | `POST` | `/api/v1/foods/:id/claim` | Protected | Klaim makanan (Database Transaction & `FOR UPDATE`) |
