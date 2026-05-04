# Dork Atölyesi (OSINT & Recon Dashboard)

Dork Atölyesi, siber güvenlik uzmanları, CTF oyuncuları ve Bug Bounty avcıları için geliştirilmiş, Go (Golang) tabanlı hızlı ve hafif bir OSINT aracıdır. Hedef domainler üzerinde pasif bilgi toplama süreçlerini hızlandırmak için özel Google Dork'ları üretir ve hedefin canlılık durumunu analiz eder.

## Özellikler

- **Dinamik Dork Üretimi:** Tek tıkla *SQL, ENV, Log dosyaları, Açık Dizinler (Index of)* ve daha fazlası için dork üretir.

- **Canlı Hedef Kontrolü (Health Check):** Arka planda Goroutines kullanarak hedefin ayakta olup olmadığını (HTTP/HTTPS) kontrol eder. WAF (Web Application Firewall - 403) korumalarını ve DNS hatalarını tespit edebilir.

- **Dışa Aktarma (Export):** Üretilen dork listelerini otomasyon araçlarında kullanmak üzere ".txt veya .json" olarak dışa aktarır.

- **Headless API Desteği:** Araç sadece web arayüzünden değil, */api/dorks?domain=hedef[.]com* ucu üzerinden terminalden veya diğer yazılımlardan da JSON formatında kullanılabilir.

- **Logger Middleware:** Yapılan tüm sorguları ve işlem sürelerini renkli olarak terminale loglar.

## Kurulum ve Çalıştırma

Proje hiçbir harici kütüphane veya framework gerektirmez. Sadece Go'nun standart kütüphaneleriyle ve arayüz için TailwindCSS (CDN) ile geliştirilmiştir.

Repoyu klonlayın ve backend dizinine gidin:

```
cd src/backend
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
src/
├── backend/
│   ├── handlers/      # API ve Form rotalarının işlendiği kontrolcüler
│   │   ├── api.go
│   │   └── dork.go
│   ├── models/        # Veri yapıları ve Dork veritabanı
│   │   └── dork_data.go
│   └── main.go        # Uygulama başlangıç noktası ve Middleware
└── frontend/
    └── index.html     # TailwindCSS ile tasarlanmış kullanıcı arayüzü
```
****
! Öğrenme ve öğretme amacıyla yapılmıştır. Yapılan sorgulardan kullanıcılar sorumludur. Geliştirici sorumlu tutulamaz. !
****