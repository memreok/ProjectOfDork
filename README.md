# Dork Atölyesi (OSINT & Recon Dashboard)

Dork Atölyesi, siber güvenlik uzmanları, CTF oyuncuları ve Bug Bounty avcıları için geliştirilmiş, Go (Golang) tabanlı hızlı ve hafif bir OSINT aracıdır. Hedef domainler üzerinde pasif bilgi toplama süreçlerini hızlandırmak için özel Google Dork'ları üretir ve hedefin canlılık durumunu analiz eder.

## Özellikler

- **Dinamik Dork Üretimi:** Tek tıkla *SQL, ENV, Log dosyaları, Açık Dizinler (Index of)* ve daha fazlası için dork üretir.

- **Canlı Hedef Kontrolü (Health Check):** Arka planda Goroutines kullanarak hedefin ayakta olup olmadığını (HTTP/HTTPS) kontrol eder. WAF (Web Application Firewall - 403) korumalarını ve DNS hatalarını tespit edebilir.

- **Dışa Aktarma (Export):** Üretilen dork listelerini otomasyon araçlarında kullanmak üzere ".txt veya .json" olarak dışa aktarır.

- **Headless API Desteği:** Araç sadece web arayüzünden değil, */api/dorks?domain=hedef[.]com* ucu üzerinden terminalden veya diğer yazılımlardan da JSON formatında kullanılabilir.

- **Logger Middleware:** Yapılan tüm sorguları ve işlem sürelerini renkli olarak terminale loglar.

- **Kalıcı Geçmiş Sorgular:** Tüm tarama sonuçları *PostgreSQL (veya Neon.tech)* üzerinde saklanır. Tekrar tarama yapmadan sonuçlar anında yüklenebilir.

## Kurulum ve Çalıştırma

### Hazırlık

Go (1.20+) ve bir PostgreSQL veritabanına (örn: Neon.tech) sahip olduğunuzdan emin olun.
Arayüz için TailwindCSS (CDN) ile geliştirilmiştir.

Repoyu klonlayın ve backend dizinine gidin:

```
git clone https://github.com/memreok/ProjectOfDork.git
```

```
cd ProjectOfDork
cd src/backend
```

Bağımlılıkların Yüklenmesi
```

go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/joho/godotenv

```

Ortam Değişkenlerini Ayarlama

src/backend dizininde .env adında bir dosya oluşturun ve veritabanı linkinizi tırnak kullanmadan ekleyin:

```
DATABASE_URL=postgresql://kullanici:sifre@sunucu-adresi/veritabani?sslmode=require
```



Projeyi çalıştırın:
```
go run main.go
```

Tarayıcınızdan arayüze erişin:
```
http://localhost:9867
```

## API Kullanımı

Terminalden (curl vb. ile) doğrudan JSON verisi almak için:
```
curl "http://localhost:9867/api/dorks?domain=ornek.com"
```

Örnek Çıktı:
```
{
  "dorks": [
    {
      "Query": "site:ornek.com intitle:\"index of\"",
      "Title": "Açık Dizinler",
      "URL": "[https://www.google.com/search?q=](https://www.google.com/search?q=)..."
    }
  ],
  "target": {
    "domain": "ornek.com",
    "is_alive": true,
    "status": "Aktif (HTTPS 200)"
  },
  "total_dorks": 10
}
```

## Proje Yapısı
```
.
├── README.md
└── src
    ├── backend
    │   ├── database
    │   │   └── db.go
    │   ├── go.mod
    │   ├── go.sum
    │   ├── handlers
    │   │   ├── api.go
    │   │   └── dork.go
    │   ├── main.go
    │   ├── models
    │   │   └── dork_data.go
    └── frontend
        └── index.html
```
****
## Yasal Uyarı

Bu araç tamamen eğitim ve güvenlik testi (OSINT) amaçlı üretilmiştir. Sorgu sonuçlarından ve kullanım şeklinden tamamen son kullanıcı sorumludur. Geliştirici, aracın kötüye kullanımından doğacak hiçbir zarardan sorumlu tutulamaz.

Geliştirici: Mehmet Emre Ök
****