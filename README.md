# Food Rescue — API (Go)

Backend REST API untuk platform Food Rescue (User, Toko, Kurir, Admin).

## Tech Stack
- Go 1.22 + Gin Framework
- MySQL 8.0+ (Hostinger, dikelola via DBeaver)
- JWT autentikasi (`golang-jwt/jwt/v4`)
- Google Sign-In via `tokeninfo` endpoint (tanpa Client ID)
- Gemini API — Asisten AI nutrisi & deteksi via kamera
- Midtrans Snap — Payment Gateway (QRIS, e-wallet, Virtual Account)

## Endpoint Lengkap

### Public (tanpa auth)
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/health` | Health check |
| POST | `/api/v1/auth/register` | Register (email+password) |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/google` | Login via Google Sign-In |
| POST | `/api/v1/auth/google/register` | Register via Google |
| GET | `/api/v1/listings` | Browse listing (filter, paginate) |
| GET | `/api/v1/listings/:id` | Detail listing |
| GET | `/api/v1/tokos` | List toko terverifikasi |
| GET | `/api/v1/tokos/:id` | Detail toko |

### Authenticated (perlu JWT Bearer)
| Method | Endpoint | Role | Fungsi |
|--------|----------|------|--------|
| GET | `/api/v1/auth/profile` | all | Profil user |
| PUT | `/api/v1/auth/profile` | all | Update profil |

#### Toko
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/tokos/me/profile` | Profil toko saya |
| PUT | `/api/v1/tokos/me/profile` | Update profil toko |
| GET | `/api/v1/tokos/me/analytics` | Analisis penjualan |
| GET | `/api/v1/tokos/me/ratings` | Lihat rating |
| POST | `/api/v1/listings` | Buat listing baru |
| GET | `/api/v1/listings/me/listings` | Listing saya |
| PUT | `/api/v1/listings/:id` | Update listing |
| DELETE | `/api/v1/listings/:id` | Hapus listing |

#### User
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/users/:id` | Profil user |
| GET | `/api/v1/users/me/impact` | Dampak personal |
| POST | `/api/v1/users/payment-methods` | Daftar metode bayar |
| GET | `/api/v1/users/payment-methods` | Daftar metode bayar |
| POST | `/api/v1/orders` | Buat order |
| GET | `/api/v1/orders/me` | Riwayat pesanan |
| GET | `/api/v1/orders/:id` | Detail pesanan |
| POST | `/api/v1/orders/:id/cancel` | Batalkan pesanan |
| POST | `/api/v1/orders/:id/confirm-payment` | Konfirmasi bayar manual |

#### Kurir
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/kurirs/me/profile` | Profil kurir |
| PUT | `/api/v1/kurirs/me/profile` | Update profil |
| PATCH | `/api/v1/kurirs/me/online` | Toggle online/offline |
| PUT | `/api/v1/kurirs/me/location` | Update lokasi |
| GET | `/api/v1/kurirs/deliveries/pending` | Tawaran masuk |
| POST | `/api/v1/kurirs/deliveries/accept` | Terima order |
| PUT | `/api/v1/kurirs/deliveries/trip` | Update status perjalanan |
| POST | `/api/v1/kurirs/deliveries/confirm-pickup` | Konfirmasi ambil barang |
| POST | `/api/v1/kurirs/deliveries/confirm-dropoff` | Konfirmasi antar barang |
| GET | `/api/v1/kurirs/me/earnings` | Riwayat pendapatan |

#### Community
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/community/` | Daftar post komunitas |
| POST | `/api/v1/community/` | Buat post (toko) |
| POST | `/api/v1/community/claim` | Klaim makanan (user) |
| POST | `/api/v1/community/:id/confirm` | Konfirmasi pengambilan |

#### Emergency
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/emergency/alerts` | Daftar alert aktif |
| GET | `/api/v1/emergency/alerts/:id/responses` | Respons toko |
| POST | `/api/v1/emergency/alerts` | Buat alert (NGO) |
| POST | `/api/v1/emergency/responses` | Respon alert (toko) |

#### Ratings, Reports, Chat, AI
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| POST | `/api/v1/ratings/` | Beri rating |
| GET | `/api/v1/ratings/:id` | Lihat rating user |
| GET | `/api/v1/ratings/me/given` | Rating yang saya berikan |
| POST | `/api/v1/reports/` | Buat laporan |
| GET | `/api/v1/reports/me` | Laporan saya |
| POST | `/api/v1/chats/` | Buat chat |
| GET | `/api/v1/chats/me` | Daftar chat saya |
| GET | `/api/v1/chats/:id/messages` | Lihat pesan |
| POST | `/api/v1/chats/:id/messages` | Kirim pesan |
| POST | `/api/v1/ai/chat` | Chat AI dari listing |
| POST | `/api/v1/ai/detect` | Deteksi nutrisi via foto |
| GET | `/api/v1/ai/conversations/:id/messages` | Riwayat chat AI |
| POST | `/api/v1/ai/conversations/:id/messages` | Lanjut chat AI |

#### Payment
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| POST | `/api/v1/payments/create` | Buat transaksi Midtrans Snap |
| POST | `/api/v1/payments/notification` | Webhook notifikasi Midtrans |

#### Admin
| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/v1/admin/dashboard` | Dashboard analitik |
| GET | `/api/v1/admin/users` | Daftar user |
| GET | `/api/v1/admin/verifications` | Menunggu verifikasi |
| POST | `/api/v1/admin/verifications/:id` | Approve/reject |
| PUT | `/api/v1/admin/users/:id/deactivate` | Nonaktifkan user |
| DELETE | `/api/v1/admin/users/:id` | Hapus user |
| PUT | `/api/v1/admin/users/:id/verify-ngo` | Verifikasi NGO |
| GET | `/api/v1/admin/reports` | Daftar laporan |
| PUT | `/api/v1/admin/reports/:id` | Review laporan |

## Setup
```bash
cp .env.example .env   # isi kredensial
go mod tidy
go run cmd/api/main.go
```

## Deploy (Docker)
```bash
docker-compose up -d --build
```
