# Panduan Deployment

Dokumen ini menjelaskan 3 metode deployment untuk Goat Chat:
1.  **Self-Hosted (Linux/STB)**: Untuk kontrol penuh dan privasi maksimal.
2.  **Render.com**: Untuk deployment awan gratis & mudah.
3.  **Koyeb**: Untuk performa WebSocket yang lebih stabil (Recommended).

---

## Opsi 1: Self-Hosted (Linux ARM64 / STB / VPS)

Cocok untuk STB B860H, Raspberry Pi, atau VPS Ubuntu.

### 1. Unduh Aplikasi
Kita akan menggunakan file *binary* siap pakai.
1.  Buka [Koleksi Rilis GitHub](https://github.com/mmuslimabdulj/goat-chat/releases).
2.  Unduh `goat-chat-linux-arm64.zip` (sesuaikan arsitektur jika pakai VPS AMD64).

### 2. Persiapan & Install
Akses server via SSH dan jalankan perintah berikut:

```bash
# 1. Buat folder
mkdir -p /opt/goat-chat

# 2. Upload file zip dari komputer lokal Anda (Jalankan di terminal PC Anda, bukan SSH)
# scp goat-chat-linux-arm64.zip root@<IP_SERVER>:/opt/goat-chat/

# 3. Ekstrak & Setup izin (Di SSH Server)
cd /opt/goat-chat
unzip goat-chat-linux-arm64.zip
cd dist
chmod +x goat-chat-linux-arm64
```

### 3. Coba Jalankan
```bash
./goat-chat-linux-arm64
# Output: GOAT chat running at http://localhost:8080
# Tekan Ctrl+C untuk stop
```

### 4. Setup Auto-Start (Systemd)
Agar aplikasi jalan otomatis saat booting:

1.  Buat file service: `nano /etc/systemd/system/goatchat.service`
2.  Paste konfigurasi ini:

    ```ini
    [Unit]
    Description=Goat Chat Server
    After=network.target

    [Service]
    Type=simple
    User=root
    WorkingDirectory=/opt/goat-chat/dist
    ExecStart=/opt/goat-chat/dist/goat-chat-linux-arm64
    Restart=always
    RestartSec=5
    
    # Opsional: Limit RAM
    # MemoryMax=500M

    [Install]
    WantedBy=multi-user.target
    ```

3.  Aktifkan service:
    ```bash
    systemctl daemon-reload
    systemctl enable --now goatchat
    ```

---

## Opsi 2: Cloud Deployment (Render.com)

Gratis, tapi aplikasi akan "tidur" setelah 15 menit inaktif.

1.  Login ke [Render.com](https://dashboard.render.com/).
2.  **New +** -> **Blueprint**.
3.  Connect repository `goat-chat`.
4.  Render akan mendeteksi Environment (Docker).
5.  Klik **Apply**.

> **Note**: Data chat akan hilang saat server restart/deploy.

---

## Opsi 3: Cloud Deployment (Koyeb)

Free tier yang bagus untuk WebSocket (koneksi lebih stabil dibanding Render).

1.  Login ke [Koyeb.com](https://www.koyeb.com/).
2.  **Create App** -> Pilih **GitHub**.
3.  Pilih repository `goat-chat`.
4.  Di bagian **Environment Variables** (Wajib):
    - `ALLOWED_ORIGINS`: `*`
    - `PORT`: `8080`
5.  Klik **Deploy**.

> **Tips**: Jangan lupa set `ALLOWED_ORIGINS` ke `*` atau domain spesifik Anda, jika tidak WebSocket akan error 403.
