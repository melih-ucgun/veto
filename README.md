# 👑 Monarch

### The Sovereign System Orchestrator

**"İşletim sistemini yönetme. Ona hükmet."**

Monarch, Linux sistem yönetimini karmaşık ve kırılgan bir süreçten; modüler, geri alınabilir ve deklaratif bir **Lego deneyimine** dönüştüren yeni nesil orkestrasyon aracıdır.

[Vizyon](https://www.google.com/search?q=%23-vizyon "null") • [Nasıl Çalışır?](https://www.google.com/search?q=%23-nas%C4%B1l-%C3%A7al%C4%B1%C5%9F%C4%B1r "null") • [Karşılaştırma](https://www.google.com/search?q=%23-neden-monarch "null") • [Yol Haritası](https://www.google.com/search?q=%23-yol-haritas%C4%B1 "null")

## 🔮 Vizyon: "Invisible OS"

Modern bir Linux kurulumu (örneğin CachyOS + Hyprland) yapmak ve korumak kaotiktir. Dotfile'lar, paketler, systemd servisleri ve kullanıcı izinleri birbirinden kopuktur. Bir şeyi değiştirdiğinizde sistem kirlenir, geri almak (Undo) neredeyse imkansızdır.

**Monarch bu kaosu bitirir.**

Sistemi tek parça bir monolit olarak değil, takılıp çıkarılabilir **Ruleset (Kural Setleri)** bütünü olarak görür.

- **Tak (Attach):** "Gaming Mode" kuralını uygula. (Steam kurulur, sürücüler ayarlanır, kernel optimize edilir.)
    
- **Sök (Detach):** Oyun oynamayı bıraktın mı? Kuralı kaldır. Monarch, kurduğu paketleri siler, değiştirdiği ayarları ve oluşturduğu dosyaları **tertemiz** bir şekilde geri alır.
    
- **Koru (Self-Heal):** Arka planda çalışan Sentinel, sistemde bir dosya manuel olarak bozulursa onu anında onarır.
    

## 🚀 Temel Özellikler

### 1. Deklaratif ve Durum Farkındalığı (State-Aware)

Monarch, körü körüne komut çalıştırmaz. Önce sistemin mevcut durumunu (`Current State`) analiz eder, hedeflediğiniz durumu (`Desired State`) ile karşılaştırır ve sadece gerekli farkı (`Diff`) uygular.

### 2. Lego Prensibi (Atomic Rulesets)

Bir uygulama sadece bir "paket" değildir. Monarch için bir _Ruleset_; paketi, konfigürasyon dosyasını, servis tanımını ve gerekli kullanıcı izinlerini içeren atomik bir bütündür.

### 3. Ajan Gerektirmez (Agentless Architecture)

Hedef sunucuda veya bilgisayarda Python, Ruby veya bir ajan kurulu olmasına gerek yoktur. Monarch, **Go** ile yazılmıştır ve tek bir binary olarak çalışır. SSH üzerinden kendini geçici olarak kopyalar, işini yapar ve iz bırakmadan silinir.

### 4. Egemenlik (Sovereignty)

Kişisel bilgisayarınızdan (Laptop), uzak sunucularınıza (VPS) kadar tüm filonuzu tek bir merkezden yönetir.

## 🆚 Neden Monarch?

Monarch; Ansible'ın gücünü, NixOS'un deterministik yapısını ve Terraform'un durum yönetimini, son kullanıcı dostu bir yapıda birleştirir.

|   |   |   |   |   |
|---|---|---|---|---|
|**Özellik**|**👑 Monarch**|**🐍 Ansible**|**❄️ NixOS**|**🐚 Shell Scripts**|
|**Dil / Hız**|**Go (Derlenmiş, Çok Hızlı)**|Python (Yavaş)|Nix (Karmaşık)|Bash (Hızlı ama güvensiz)|
|**Geri Alma (Undo)**|✅ **Native (Otomatik)**|❌ Yok (Manuel)|✅ (Rollback)|❌ Yok|
|**Durum Takibi**|✅ **State.json + Checksum**|❌ Kısıtlı (Facts)|✅ (Store)|❌ Yok|
|**Bağımlılık**|**Yok (Single Binary)**|Python gerektirir|Özel OS gerektirir|Bağımlılık Cehennemi|
|**Öğrenme Eğrisi**|**Düşük (Lego Mantığı)**|Orta (YAML karmaşası)|Çok Yüksek|Değişken|
|**Kullanım**|Desktop & Server|Server Odaklı|Tüm OS|Basit işler|

## 🏗️ Mimari: Kutsal Üçlü

Monarch ekosistemi üç ana sütun üzerine inşa edilmektedir:

1. **Monarch Engine (CLI):** Sistemin beyni. Go ile yazılmış, `resource`, `apply`, `diff` mantığını yürüten çekirdek.
    
2. **Monarch Hub (The Library):** GitHub tabanlı global kural kütüphanesi. Başkalarının hazırladığı "Hyprland Setup" veya "DevOps Stack" kurallarını tek komutla çekebileceğiniz yer.
    
3. **Monarch Studio (GUI):** Terminal korkusunu yenen, Wails ile geliştirilecek modern masaüstü arayüzü. Sistemi bir kokpit gibi yönetmenizi sağlar.
    

## 🛠️ Teknoloji Yığını

- **Core:** [Go (Golang)](https://go.dev/ "null") - Yüksek performans ve concurrency.
    
- **Config:** YAML - İnsan tarafından okunabilir, basit yapı.
    
- **State:** JSON - Taşınabilir ve hafif durum takibi.
    
- **Security:** [Age (X25519)](https://github.com/FiloSottile/age "null") - Modern ve güvenli secret (şifre) yönetimi.
    
- **Transport:** SSH - Güvenli uzak sunucu yönetimi.
    

## ⚡ Hızlı Başlangıç (Alpha)

Monarch şu an geliştirme aşamasındadır. Denemek için:

```
# 1. Depoyu klonlayın
git clone [https://github.com/melih-ucgun/monarch.git](https://github.com/melih-ucgun/monarch.git)
cd monarch

# 2. Derleyin
go build -o monarch main.go

# 3. Örnek bir konfigürasyonu uygulayın (Dry-Run)
./monarch apply --config monarch.yaml --dry-run
```

### Örnek `monarch.yaml`

```
resources:
  - type: package
    id: neovim
    name: neovim
    state: present

  - type: file
    id: nvim-config
    path: ~/.config/nvim/init.lua
    content: |
      print("Hello from Monarch Managed Config!")
    owner: melih
    mode: "0644"

  - type: service
    id: docker-service
    name: docker
    state: running
    enabled: true
```

## 🗺️ Yol Haritası

Monarch sürekli gelişiyor. İşte planımız:

- [x] **Çekirdek (Hazır):** Temel komutlar, dosya/paket yönetimi ve durum takibi.
    
- [ ] **Geri Al & Hub (Sıradaki):** `Undo` özelliği ve GitHub entegrasyonu.
    
- [ ] **Arayüz (GUI):** Modern masaüstü uygulaması ve Hyprland entegrasyonu.
    
- [ ] **Otonom:** Kendi kendini onaran (Self-healing) sistem ve filo yönetimi.
    

**Monarch** © 2025 Melih Uçgun tarafından, kontrol manyakları ve sistem mimarları için ❤️ ile geliştirildi.
