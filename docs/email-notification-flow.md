# Alur Email Notifikasi Perubahan Status Order

Dokumen ini menjelaskan bagaimana email notifikasi dikirim saat status order berubah, kenapa arsitektur memakai Kafka, dan apa dampaknya jika message broker tidak digunakan.

## Ringkasan Arsitektur

Komponen utama:
- `Order Service` (producer event)
- `Outbox Worker` (publisher ke Kafka)
- `Kafka` (message broker)
- `Consumer` (pemroses event + pengirim email)
- `Email Service` (Resend)

Event yang terkait email:
- `ORDER_STATUS_CHANGED`
- `ORDER_PAYMENT_UPDATED`

## Alur Kirim Email Saat Status Order Berubah

1. Admin/customer melakukan aksi update status order melalui API.
2. `order_service` update status di database dalam transaksi.
3. Masih di transaksi yang sama, service membuat event ke tabel outbox (`ORDER_STATUS_CHANGED`).
4. Worker membaca outbox dan mem-publish event ke topic Kafka (`order.events`) lalu menandai outbox sebagai terkirim.
5. `consumer` subscribe topic `order.events`, membaca header `event_type`.
6. Jika `event_type=ORDER_STATUS_CHANGED`, consumer:
   - parse payload
   - ambil data user dari DB
   - panggil `emailSvc.SendOrderStatusEmail(...)`
7. Jika sukses, offset Kafka di-commit agar event tidak diproses ulang.

## Kenapa Pakai Kafka

Manfaat utama:
- Decoupling: proses order tidak tergantung langsung pada layanan email.
- Reliability: event bisa diproses ulang jika consumer gagal sementara.
- Durability: pesan tersimpan di broker, tidak hilang saat service restart.
- Scalability: consumer bisa ditambah (horizontal) saat traffic naik.
- Backpressure handling: lonjakan request tidak langsung membebani API/email provider.
- Observability: event stream mudah di-audit dan ditrace.

## Peran Outbox Pattern

Outbox memastikan konsistensi antara perubahan data order dan event:
- Status order berubah dan event tercatat dalam **satu transaksi DB**.
- Mencegah kasus status sudah berubah tapi event tidak pernah terkirim.
- Worker menangani retry publish secara asynchronous.

## Efek Jika Tidak Pakai Message Broker

Jika email dipanggil sinkron langsung dari request update status:
- Latency API naik karena menunggu call ke provider email.
- Kegagalan email bisa membuat request gagal atau butuh kompensasi rumit.
- Risk timeout lebih tinggi saat provider lambat.
- Sulit retry terkontrol tanpa bikin duplikasi.
- Tight coupling: logic order menjadi bergantung ke layanan eksternal.
- Ketika traffic tinggi, API lebih mudah overload.

Jika tanpa broker tapi tetap async in-process (goroutine lokal):
- Event hilang saat proses crash/restart.
- Tidak ada persistence queue.
- Tidak ada consumer group/offset management.

## Failure Mode yang Perlu Diperhatikan

- `RESEND_API_KEY` tidak tersedia: consumer fallback ke `NoopService` (log terlihat sukses, email tidak benar-benar terkirim).
- Kafka down: worker/consumer tertahan; outbox akan menumpuk.
- DB user lookup gagal: consumer retry (offset belum commit).
- Email provider error: consumer retry event yang sama.

## Checklist Operasional

- Pastikan `consumer` punya env:
  - `RESEND_API_KEY`
  - `RESEND_FROM_EMAIL`
- Pastikan worker dan consumer aktif.
- Monitor:
  - ukuran backlog outbox
  - consumer lag Kafka
  - error rate pengiriman email
- Buat alert jika fallback ke `NoopService` terdeteksi di log.

