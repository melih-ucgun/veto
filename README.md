👑 Monarch"The Sovereign Infrastructure Orchestrator"Monarch; yerel makineler, uzak sunucular ve hibrit altyapılar için tasarlanmış, Go ile geliştirilen, ajan gerektirmeyen (agentless) ve durum tabanlı (declarative) bir sistem yönetim aracıdır.Sistem yönetimini basit script'lerden çıkarıp, sistemin olması gereken halini tanımladığınız bir mimariye dönüştürür. Ansible'ın esnekliğini ve Go'nun hızını tek bir binary dosyasında birleştirir.🔥 Temel Özellikler🚀 Yüksek Performans: Go (Golang) ile yazılmış, hafif ve hızlı.📦 Tek Binary: Bağımlılık gerektirmez, sadece çalıştırılabilir dosyayı taşımanız yeterlidir.🛠️ Deklaratif Yapı: "Nasıl yapılacağını" değil, "ne olması gerektiğini" tanımlayın.🔒 Güvenli Sır Yönetimi: age kütüphanesi ile entegre şifrelenmiş veri yönetimi.🏗️ Geniş Kaynak Desteği:Package: Sistem paketlerini yönetin (Pacman, Apt adaptörleri).File & Template: Dosya taşıma ve dinamik şablonlama.Service: Systemd servislerini kontrol edin.Archive: Uzak URL'lerden .tar.gz veya .zip indirip otomatik açın (Yeni!).Git & Container: Depo yönetimi ve Podman/Docker desteği.Exec & Symlink: Özel komutlar ve sembolik linkler.🛠️ KurulumMonarch'ı yerelinizde derlemek için Go (1.21+) yüklü olmalıdır:# Depoyu klonlayın
git clone [https://github.com/melih-ucgun/monarch](https://github.com/melih-ucgun/monarch)
cd monarch

# Bağımlılıkları indirin ve derleyin
go mod tidy
go build -o monarch main.go

# Global kullanım için (opsiyonel)
sudo mv monarch /usr/local/bin/
📖 Hızlı BaşlangıçMonarch, sistem durumunu YAML dosyaları üzerinden okur. Örnek bir kurulum (v0.1.0-alpha):1. Yapılandırma Oluşturun (monarch.yaml)inventory:
  - name: "local-machine"
    host: "localhost"
    user: "user"

resources:
  - name: "install-micro"
    archive:
      source: "[https://github.com/zyedidia/micro/releases/download/v2.0.14/micro-2.0.14-linux64.tar.gz](https://github.com/zyedidia/micro/releases/download/v2.0.14/micro-2.0.14-linux64.tar.gz)"
      destination: "/usr/local/bin"
      strip_components: 1
      check_file: "micro"

  - name: "ensure-config-dir"
    exec:
      command: "mkdir -p ~/.config/monarch"
      check: "test -d ~/.config/monarch"
2. Uygulayın./monarch apply -c monarch.yaml
🔐 Sır Yönetimi (Secrets)Hassas verilerinizi düz metin olarak saklamayın. Monarch'ın yerleşik şifreleme özelliğini kullanın:# Bir veriyi şifrele
./monarch secrets encrypt "hassas_verim"

# Şifrelenmiş veriyi YAML içinde kullanın
# Monarch uygulama sırasında bu veriyi otomatik olarak çözecektir.
🗺️ Yol Haritası (Roadmap)[ ] Dconf/GSettings: Masaüstü ortamı ayarları için destek (Hyprland/GNOME).[ ] Flatpak: Sandbox uygulama yönetimi.[ ] Firewall: UFW/NFTables deklaratif yönetimi.[ ] Gelişmiş Diff: Değişiklikleri uygulamadan önce görselleştirme.⚠️ Alpha Sürüm NotuBu proje şu anda v0.1.0-alpha aşamasındadır. Temel özellikler stabil çalışmakla birlikte, kritik üretim sistemlerinde kullanmadan önce yapılandırmalarınızı test etmeniz önerilir.📄 LisansBu proje AGPL 3.0 Lisansı ile lisanslanmıştır. Daha fazla bilgi için LICENSE dosyasına bakınız.Developed with ❤️ for the Linux community.
