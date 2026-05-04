package models

type DorkItem struct {
	Title       string
	Description string
	Example     string
}

type GeneratedDork struct {
	Title string
	Query string
	URL   string
}

type PageData struct {
	Domain          string
	SelectedDork    string
	Results         []GeneratedDork
	Dorks           []DorkItem
	ErrorMessage    string
	IsTargetAlive   bool
	TargetStatusMsg string
}

var DorkLibrary = []DorkItem{
	{"Açık Dizinler", "Sunucuda unutulmuş klasörleri listeler.", "intitle:\"index of\""},
	{"Veritabanı Sızıntıları", "Dışarıya sızmış SQL yedeklerini arar.", "ext:sql intext:password"},
	{"Çevre Değişkenleri", ".env dosyalarındaki hassas verileri bulur.", "ext:env intext:DB_PASSWORD"},
	{"Log Dosyaları", "Sistem günlüklerini ve hata mesajlarını hedefler.", "ext:log intext:error"},
	{"Açık Git Klasörleri", "Public unutulmuş .git dizinlerini bulur.", "inurl:\"/.git\" -github.com"},
	{"WordPress Config", "WP wp-config.php dosyalarını arar.", "inurl:wp-config.php intext:DB_PASSWORD"},
	{"SSH Anahtarları", "Yanlışlıkla public olmuş SSH anahtarlarını bulur.", "intitle:\"index of\" id_rsa"},
	{"Trello & Jira Pano", "Şirket içi açık unutulmuş panoları arar.", "inurl:trello.com/b | inurl:jira"},
	{"S3 Bucket Sızıntıları", "Hatalı yapılandırılmış Amazon S3 depoları.", "site:s3.amazonaws.com"},
	{"API Endpoint'leri", "Açıkta kalmış API dökümanları (Swagger vb.)", "inurl:api/v1 | inurl:swagger"},
}
