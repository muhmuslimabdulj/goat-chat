# Panduan Setup Android Development

Panduan ini menjelaskan cara **build dan run aplikasi Android di WSL2/Linux tanpa Android Studio**.

## Mulai Cepat

### 1. Setup Awal (Satu Kali)
Install SDK, dependencies, dan buat emulator:
```bash
bash scripts/android/setup_android.sh
source ~/.bashrc
```

### 2. Build APK
```bash
bash scripts/android/build_apk.sh
```

### 3. Install ke HP Android
Hubungkan HP via USB (pastikan USB Debugging aktif).

**Windows (Pengguna WSL):**
Buka PowerShell di Windows:
```powershell
cd \\wsl$\Ubuntu\home\<username>\code\goat-chat
adb install -r android\app\build\outputs\apk\debug\app-debug.apk
```

**Linux (Native / Bare Metal):**
Jika menggunakan Linux langsung (bukan WSL), pastikan `adb` sudah terinstall (`sudo apt install adb`), lalu jalankan:
```bash
adb install -r android/app/build/outputs/apk/debug/app-debug.apk
```

---

## Persyaratan Sistem

| Requirement | Keterangan |
|-------------|------------|
| **OS**      | WSL2 (Windows 10/11) atau Linux (Ubuntu 22.04+) |
| **Node.js** | v22+ (via nvm, pnpm, atau apt) |
| **Disk**    | ~5GB untuk Android SDK dan system images |
| **RAM**     | Minimal 8GB (16GB recommended untuk emulator) |

Script `setup_android.sh` akan otomatis install:
- OpenJDK 21 (Diperlukan oleh Capacitor 8)
- Android SDK Command Line Tools
- Platform 34, Build Tools 34.0.0
- Android Emulator & System Image (x86_64)
- GUI dependencies (libxcb, libpulse, dll)

---

## Workflow Development

### 1. Development Sehari-hari (Web)
```bash
make dev                    
# Start server dengan hot-reload
# Buka browser: http://localhost:8080
```
Ini cara tercepat untuk development — edit code, lihat perubahan langsung di browser.

### 2. Build APK
```bash
bash scripts/android/build_apk.sh
```
Output: `android/app/build/outputs/apk/debug/app-debug.apk`

### 3. Test di HP Android
HP harus terhubung via USB dengan **USB Debugging** aktif.

**Di Windows PowerShell:**
```powershell
# Masuk ke folder project via WSL path
cd \\wsl$\Ubuntu\home\<username>\code\goat-chat

# Install APK
adb install -r android\app\build\outputs\apk\debug\app-debug.apk
```

### 4. Test di Emulator (Opsional)
```bash
bash scripts/android/run_emulator.sh
```
> **Catatan:** Emulator di WSL2 menggunakan software rendering (lambat). Untuk testing yang lebih nyaman, gunakan HP asli.

---

## Referensi Script

Semua script ada di `scripts/android/`:

| Script | Deskripsi |
|--------|-----------|
| `setup_android.sh` | One-time setup: Install SDK, dependencies, buat AVD |
| `build_apk.sh` | Build debug APK dengan production URL |
| `build_apk_dev.sh` | Build debug APK dengan localhost (untuk dev) |
| `run_emulator.sh` | Start emulator, install APK, launch app |
| `run_emulator_headless.sh` | Start emulator tanpa GUI (untuk CI/CD) |
| `run_device.sh` | Deploy ke HP via USB/TCP |

---

## Troubleshooting

### Emulator window tidak muncul
**Penyebab:** WSLg (GUI subsystem) perlu di-restart.

**Solusi:** Di Windows PowerShell (bukan WSL):
```powershell
wsl --shutdown
```
Tunggu 10 detik, buka terminal WSL lagi, jalankan ulang emulator.

### KVM permission denied
**Penyebab:** User tidak punya akses ke hardware virtualization.

**Solusi:**
```bash
sudo chmod 666 /dev/kvm
```
Atau tambahkan user ke group kvm (perlu logout/login ulang):
```bash
sudo usermod -aG kvm $USER
```

### "Command not found" untuk adb/emulator
**Penyebab:** Environment variables belum dimuat.

**Solusi:**
```bash
source ~/.bashrc
```
Atau restart terminal.

### HP Android tidak terdeteksi di adb
1. Pastikan **USB Debugging** aktif di HP:
   - Settings → Developer Options → USB Debugging: ON
2. Saat colok USB, pilih **Allow** pada popup "Trust this computer?"
3. Pastikan mode USB adalah **File Transfer (MTP)**, bukan Charging only
4. Coba `adb kill-server && adb start-server`

### APK crash atau blank screen
- Jika menggunakan `build_apk_dev.sh` (localhost): APK hanya berfungsi jika server development berjalan dan device dapat mengakses IP server
- Gunakan `build_apk.sh` (production) untuk testing yang stabil

---

## Capacitor Cheatsheet

Daftar perintah penting untuk pengelolaan proyek mobile:

| Perintah | Fungsi |
|----------|--------|
| `npx cap sync` | Sinkronisasi aset web (public/dist) & config ke folder android |
| `npx cap open android` | Membuka proyek di Android Studio (jika terinstall) |
| `npx cap run android` | Run aplikasi di device/emulator yang terhubung |
| `npx cap add android` | Menambahkan platform Android (inisialisasi ulang) |
| `make build` | Build aset web sebelum sync (penting!) |

> **Tips:** Selalu jalankan `make build` sebelum `npx cap sync` agar perubahan kode web terbaru ikut ter-update di aplikasi Android.

---

## Lokasi File Penting

| File | Path |
|------|------|
| **Debug APK** | `android/app/build/outputs/apk/debug/app-debug.apk` |
| **Android SDK** | `~/android-sdk/` |
| **AVD** | `~/.android/avd/pixel_5_api_34.avd/` |
| **Emulator Log** | `emulator.log` (di folder project) |
| **Capacitor Config** | `capacitor.config.ts` |

---

## FAQ

**Q: Apakah saya perlu Android Studio?**
A: Tidak. Semua yang dibutuhkan sudah terinstall via `setup_android.sh`.

**Q: Berapa lama proses setup?**
A: Sekitar 10-15 menit untuk download SDK (~2GB) dan system image (~1.5GB).

**Q: Kenapa local development ke HP tidak bisa?**
A: WSL2 punya IP internal yang tidak bisa diakses device external secara langsung. Gunakan production build untuk testing di HP.

**Q: Bisa build release/signed APK?**
A: Script ini untuk debug APK. Untuk release, perlu konfigurasi keystore di `android/app/build.gradle`.

---

## CI/CD (GitHub Actions)

Proyek ini memiliki workflow otomatis `.github/workflows/android.yml` untuk build APK di cloud.

### Konfigurasi URL Server

Secara default, CI akan menggunakan URL Production: `https://goat-chat.koyeb.app`.
Untuk mengubahnya (misal ke staging), Anda punya 2 opsi:

#### Opsi 1: Menggunakan GitHub Secrets (Permanen)
Ini disarankan agar tidak perlu input URL setiap kali build.

1. Buka repository di GitHub.
2. Masuk ke **Settings** > **Secrets and variables** > **Actions**.
3. Klik **New repository secret**.
4. Isi:
   - **Name**: `CAPACITOR_SERVER_URL`
   - **Secret**: URL server Anda (contoh: `https://staging-goat.koyeb.app`)
5. Klik **Add secret**.

#### Opsi 2: Manual Trigger (Sekali Pakai)
Jika ingin build khusus tanpa mengubah secret:

1. Pergi ke tab **Actions**.
2. Pilih workflow **Build Android APK**.
3. Klik **Run workflow**.
4. Isi input **Server URL for the app**.
5. Klik **Run workflow**.

