# RabbitMQ Package Documentation

Package `rabbitmq` ini menyediakan wrapper untuk mengelola koneksi RabbitMQ menggunakan `amqp091-go`. Package ini sudah dilengkapi dengan fitur *auto-reconnect*, *retry on startup*, serta otomatisasi setup topologi (Exchange, Queue, dan Binding) langsung dari konfigurasi.

## 1. Struktur Konfigurasi (YAML)

 berikut adalah bentuk implementasinya jika ditulis dalam format file YAML (misalnya menggunakan Viper untuk membaca konfigurasinya). Contoh di bawah ini mengambil studi kasus implementasi *chat-app*:

```yaml
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  max_channels: 100
  reconnect_delay: 5s
  prefetch_count: 10
  prefetch_size: 0
  heartbeat: 10s
  connection_name: "chat-app"
  
  # Daftar Exchange yang akan di-declare secara otomatis
  exchanges:
    - name: "chat.messages"
      kind: "topic"          # Bisa direct, topic, fanout, atau headers
      durable: true
      auto_delete: false
      internal: false
      no_wait: false
      arguments: null        # Format map (key-value) jika ada args tambahan

  # Daftar Queue dan Bindings yang akan di-declare secara otomatis
  queues:
    - name: "chat.inbox.queue"
      durable: true
      auto_delete: false
      exclusive: false
      no_wait: false
      arguments:             # Contoh penambahan argument (misal: DLQ)
        "x-dead-letter-exchange": "chat.dlx"
      
      # Konfigurasi binding queue ke exchange
      bindings:
        - name: "bind_inbox_messages" # Penanda identifier (opsional di level config)
          key: "chat.message.*"       # Routing key
          exchange: "chat.messages"   # Nama exchange tujuan
          no_wait: false
          arguments: null
```

2. Penjelasan Topology Declaration
Saat koneksi berhasil dibuat, package akan memanggil fungsi setupTopology() yang bertugas mendeklarasikan seluruh struktur antrean. Berikut adalah penjelasan untuk masing-masing tahapannya:

A. Exchange Declaration (ExchangeDeclare)
Package akan melakukan iterasi pada array exchanges dari konfigurasi dan mendaftarkannya ke server RabbitMQ. Parameter yang digunakan:

name: Nama dari exchange (misal: chat.messages).

kind: Tipe distribusi pesan (direct, topic, fanout, headers).

durable: Jika true, exchange akan tetap ada meskipun server RabbitMQ di-restart (disimpan ke disk).

auto_delete: Jika true, exchange akan otomatis dihapus ketika tidak ada lagi queue yang terhubung dengannya.

internal: Jika true, exchange ini tidak bisa menerima pesan langsung dari publisher, melainkan hanya menerima pesan dari exchange lain.

no_wait: Jika true, server tidak akan mengirimkan respons sukses ke klien (fire and forget).

arguments: Argumen tambahan (misal untuk konfigurasi Alternate Exchange).

B. Queue Declaration (QueueDeclare)
Selanjutnya, package melakukan iterasi pada array queues. Parameter yang digunakan:

name: Nama unik antrean (misal: chat.inbox.queue).

durable: Jika true, antrean (bukan isinya) akan bertahan saat RabbitMQ restart.

auto_delete: Jika true, queue akan terhapus otomatis ketika tidak ada lagi consumer yang menggunakannya.

exclusive: Jika true, antrean ini hanya bisa diakses oleh koneksi yang membuatnya dan akan dihapus saat koneksi tersebut terputus.

no_wait: Jika true, deklarasi berjalan secara asinkron tanpa menunggu konfirmasi server.

arguments: Sangat berguna untuk mendefinisikan Dead Letter Exchange (DLX), Message TTL, atau Queue Length Limit.

C. Queue Binding (QueueBind)
Di dalam setiap konfigurasi Queue, terdapat array Bindings. Tahap ini menyambungkan Queue yang baru saja dibuat ke Exchange yang sudah ada. Parameter yang digunakan:

queue_name: Nama queue yang sedang di-iterasi.

key: Routing key yang menentukan aturan pesan mana yang boleh masuk ke queue ini (misal chat.message.*).

exchange: Nama exchange tujuan tempat queue ini bergantung.

no_wait: Menjalankan binding tanpa menunggu konfirmasi dari server.

arguments: Aturan tambahan yang biasa digunakan jika jenis exchange adalah headers.


3. Fitur Utama Package
Retry Mechanism & Auto-Reconnect:

Jika server RabbitMQ belum siap saat startup, package akan melakukan retry hingga batas maxRetries (10 kali) dengan jeda sesuai reconnect_delay.

Memiliki goroutine handleReconnect() yang akan secara otomatis melakukan koneksi ulang dan me-restore topology jika koneksi terputus tiba-tiba (seperti network drop).

Quality of Service (QoS) Otomatis:

Pada saat channel dibuat, PrefetchCount dan PrefetchSize diatur secara global untuk mencegah consumer kewalahan menerima lonjakan pesan.

Graceful Shutdown:

Method Close() memastikan channel internal (r.done) dan seluruh koneksi TCP ke RabbitMQ ditutup dengan bersih untuk mencegah memory leak atau zombie connections.