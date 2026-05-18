package models

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	DefaultCustomDork = "ext:sql | ext:env"
	domainToken       = "{domain}"
)

type DorkItem struct {
	ID          string
	Category    string
	Title       string
	Description string
	Example     string
	Tags        []string
}

type GeneratedDork struct {
	Title       string
	Category    string
	Description string
	Query       string
	URL         string
}

type HistoryItem struct {
	ID              uint   `gorm:"primaryKey"`
	Domain          string `gorm:"index"`
	IsTargetAlive   bool
	TargetStatusMsg string
	Timestamp       time.Time `gorm:"autoCreateTime"`
}

type PageData struct {
	Domain          string
	SelectedDork    string
	Results         []GeneratedDork
	DorkList        []string
	Dorks           []DorkItem
	Categories      []string
	History         []HistoryItem
	ErrorMessage    string
	IsTargetAlive   bool
	TargetStatusMsg string
}

var DorkLibrary = normalizeDorkLibrary([]DorkItem{
	{ID: "open-directories", Category: "Dosya ve Dizin", Title: "Açık Dizinler", Description: "Sunucuda unutulmuş dizin listelemelerini bulur.", Example: `intitle:"index of"`, Tags: []string{"index-of", "directory"}},
	{ID: "database-leaks", Category: "Veritabanı", Title: "Veritabanı Sızıntıları", Description: "Dışarıya sızmış SQL yedeklerini arar.", Example: `ext:sql intext:password`, Tags: []string{"sql", "backup"}},
	{ID: "env-files", Category: "Kimlik Bilgileri", Title: "Çevre Değişkenleri", Description: ".env dosyalarındaki hassas yapılandırmaları bulur.", Example: `ext:env intext:DB_PASSWORD`, Tags: []string{"env", "secret"}},
	{ID: "log-files", Category: "Hata ve Log", Title: "Log Dosyaları", Description: "Sistem günlüklerini ve hata mesajlarını hedefler.", Example: `ext:log intext:error`, Tags: []string{"log", "error"}},
	{ID: "exposed-git", Category: "Kaynak Kod", Title: "Açık Git Klasörleri", Description: "Public unutulmuş .git dizinlerini bulur.", Example: `inurl:"/.git" -github.com`, Tags: []string{"git", "source"}},
	{ID: "wp-config", Category: "CMS", Title: "WordPress Config", Description: "Açığa çıkmış wp-config.php dosyalarını arar.", Example: `inurl:wp-config.php intext:DB_PASSWORD`, Tags: []string{"wordpress", "config"}},
	{ID: "ssh-keys", Category: "Kimlik Bilgileri", Title: "SSH Anahtarları", Description: "Yanlışlıkla public olmuş SSH anahtarlarını bulur.", Example: `intitle:"index of" id_rsa`, Tags: []string{"ssh", "private-key"}},
	{ID: "project-boards", Category: "SaaS ve Pano", Title: "Trello ve Jira Panoları", Description: "Şirket içi açık unutulmuş panoları arar.", Example: `inurl:trello.com/b | inurl:jira`, Tags: []string{"trello", "jira"}},
	{ID: "s3-buckets", Category: "Bulut", Title: "S3 Bucket Sızıntıları", Description: "Hatalı yapılandırılmış Amazon S3 depolarını arar.", Example: `site:s3.amazonaws.com "{domain}"`, Tags: []string{"aws", "s3"}},
	{ID: "api-docs", Category: "API", Title: "API Endpoint'leri", Description: "Açıkta kalmış API dökümanlarını ve Swagger panellerini bulur.", Example: `inurl:api/v1 | inurl:swagger`, Tags: []string{"api", "swagger"}},
	{ID: "bug-bounty-pages", Category: "Recon", Title: "VDP ve Bug Bounty Programları", Description: "Güvenlik bildirimi veya bug bounty sayfalarını bulur.", Example: `intext:"submit your vulnerability" intitle:"bug bounty"`, Tags: []string{"vdp", "bug-bounty"}},
	{ID: "security-txt", Category: "Recon", Title: "Security.txt Dosyaları", Description: "RFC 9116 güvenlik iletişim dosyalarını arar.", Example: `inurl:.well-known/security.txt`, Tags: []string{"security.txt", "policy"}},
	{ID: "zip-directory", Category: "Dosya ve Dizin", Title: "Açık Dizinler (ZIP)", Description: "Dizin listelemeye açık ZIP arşivlerini bulur.", Example: `intitle:"index of /" filetype:zip`, Tags: []string{"zip", "backup"}},
	{ID: "apache-logs", Category: "Hata ve Log", Title: "Apache Sunucu Logları", Description: "Açıkta kalmış Apache sunucu günlüklerini arar.", Example: `intitle:"Apache" filetype:log`, Tags: []string{"apache", "log"}},
	{ID: "nginx-index", Category: "Dosya ve Dizin", Title: "Nginx Dizin Listeleme", Description: "Hatalı yapılandırılmış Nginx dizin listelemelerini arar.", Example: `intitle:"Nginx" intitle:"Index of /"`, Tags: []string{"nginx", "index-of"}},
	{ID: "admin-html", Category: "Yönetim Paneli", Title: "Yönetici HTML Panelleri", Description: "Açıkta kalmış admin giriş formlarını bulur.", Example: `inurl:admin filetype:html`, Tags: []string{"admin", "login"}},
	{ID: "administrator-paths", Category: "Yönetim Paneli", Title: "Administrator Klasörleri", Description: "Yönetici alt alan adları veya dizinlerini tarar.", Example: `inurl:administrator`, Tags: []string{"admin", "path"}},
	{ID: "cpanel-login", Category: "Yönetim Paneli", Title: "cPanel Girişleri", Description: "Paylaşımlı sunuculardaki cPanel panellerini bulur.", Example: `inurl:cpanel`, Tags: []string{"cpanel", "login"}},
	{ID: "staging-envs", Category: "Recon", Title: "Staging ve Test Ortamları", Description: "Korumasız test, dev ve beta ortamlarını arar.", Example: `inurl:staging OR inurl:test OR inurl:dev OR inurl:beta`, Tags: []string{"staging", "dev"}},
	{ID: "old-apache-version", Category: "Teknoloji İfşası", Title: "Eski Apache Versiyonları", Description: "Belirli Apache sürüm ifşalarını arar.", Example: `intext:"Apache/2.4.1"`, Tags: []string{"apache", "version"}},
	{ID: "sql-errors", Category: "Hata ve Log", Title: "SQL Sözdizimi Hataları", Description: "SQL Injection riskine işaret eden hata mesajlarını bulur.", Example: `intext:"SQL syntax error"`, Tags: []string{"sql", "error"}},
	{ID: "stack-trace", Category: "Hata ve Log", Title: "Stack Trace İfşaları", Description: "Backend mimarisini ve hata yollarını sızdıran sayfaları arar.", Example: `intext:"Stack trace"`, Tags: []string{"stack-trace", "error"}},
	{ID: "mysql-warning", Category: "Hata ve Log", Title: "MySQL Hata İfşası", Description: "Veritabanı yapısını sızdıran MySQL hatalarını hedefler.", Example: `intext:"Warning: mysql_fetch_array()"`, Tags: []string{"mysql", "error"}},
	{ID: "fatal-error", Category: "Hata ve Log", Title: "Fatal Error Path İfşası", Description: "Sunucu iç yolunu gösteren ölümcül hataları arar.", Example: `intext:"Fatal error in"`, Tags: []string{"path", "error"}},
	{ID: "api-key", Category: "Kimlik Bilgileri", Title: "API Key Sızıntıları", Description: "Sayfa içeriğine gömülü API anahtarlarını arar.", Example: `intext:"api_key="`, Tags: []string{"api-key", "secret"}},
	{ID: "tokens", Category: "Kimlik Bilgileri", Title: "Gizli Jetonlar", Description: "Kimlik doğrulama token sızıntılarını arar.", Example: `intext:"token="`, Tags: []string{"token", "secret"}},
	{ID: "aws-keys", Category: "Bulut", Title: "AWS Access Key Sızıntısı", Description: "AWS erişim anahtarı veya secret ifadelerini arar.", Example: `intext:"AKIA" OR intext:"aws_secret_access_key"`, Tags: []string{"aws", "secret"}},
	{ID: "google-api-key", Category: "Kimlik Bilgileri", Title: "Google API Anahtarları", Description: "JavaScript veya config içinde unutulmuş Google API keylerini arar.", Example: `intext:"google_api_key"`, Tags: []string{"google", "api-key"}},
	{ID: "rsa-private-key", Category: "Kimlik Bilgileri", Title: "RSA Private Key", Description: "İfşa olmuş RSA gizli anahtarlarını bulur.", Example: `"-----BEGIN RSA PRIVATE KEY-----"`, Tags: []string{"rsa", "private-key"}},
	{ID: "openssh-private-key", Category: "Kimlik Bilgileri", Title: "OpenSSH Private Key", Description: "Açıkta kalmış OpenSSH özel anahtarlarını hedefler.", Example: `"-----BEGIN OPENSSH PRIVATE KEY-----"`, Tags: []string{"openssh", "private-key"}},
	{ID: "pem-files", Category: "Sertifika", Title: "PEM Sertifikaları", Description: "PEM formatlı sertifika ve anahtar dosyalarını arar.", Example: `filetype:pem`, Tags: []string{"pem", "certificate"}},
	{ID: "confidential-pdf", Category: "Belge", Title: "İç Belgeler (Gizli PDF)", Description: "Confidential veya internal etiketli PDF belgeleri arar.", Example: `filetype:pdf "internal" "confidential"`, Tags: []string{"pdf", "internal"}},
	{ID: "password-xlsx", Category: "Belge", Title: "Parola İçeren Excel Dosyaları", Description: "Şifre veya token barındıran elektronik tabloları arar.", Example: `filetype:xlsx "password" OR "secret" OR "token"`, Tags: []string{"xlsx", "password"}},
	{ID: "backup-files", Category: "Dosya ve Dizin", Title: "BAK Yedek Dosyaları", Description: "Eski yapılandırma ve uygulama yedek dosyalarını arar.", Example: `filetype:bak OR filetype:backup`, Tags: []string{"backup", "bak"}},
	{ID: "sql-dumps", Category: "Veritabanı", Title: "Açık SQL Veritabanı Dökümleri", Description: "İnternete açık SQL dump dosyalarını ve tablolarını hedefler.", Example: `filetype:sql intext:"CREATE TABLE"`, Tags: []string{"sql", "dump"}},
	{ID: "github-npmrc", Category: "GitHub", Title: "GitHub: NPM Auth Token", Description: "NPM registry yetki bilgilerini GitHub'da bulur.", Example: `site:github.com "{domain}" ".npmrc" "_auth"`, Tags: []string{"npm", "token"}},
	{ID: "github-dockercfg", Category: "GitHub", Title: "GitHub: Docker Config", Description: "Docker registry kimlik bilgilerinin GitHub sızıntılarını arar.", Example: `site:github.com "{domain}" ".dockercfg" "auth"`, Tags: []string{"docker", "auth"}},
	{ID: "github-ppk", Category: "GitHub", Title: "GitHub: PPK Key", Description: "PuttyGen özel anahtarlarını kod depolarında arar.", Example: `site:github.com "{domain}" "BEGIN PPK PRIVATE KEY"`, Tags: []string{"ppk", "private-key"}},
	{ID: "github-s3cfg", Category: "GitHub", Title: "GitHub: S3 Yapılandırması", Description: "S3 CLI araçlarına ait AWS konfigürasyonlarını arar.", Example: `site:github.com "{domain}" ".s3cfg"`, Tags: []string{"aws", "s3"}},
	{ID: "github-htpasswd", Category: "GitHub", Title: "GitHub: HTPasswd Dosyaları", Description: "Dizin koruma şifre hash'lerini hedefler.", Example: `site:github.com "{domain}" ".htpasswd"`, Tags: []string{"htpasswd", "hash"}},
	{ID: "github-laravel-env", Category: "GitHub", Title: "GitHub: Laravel Çevre Değişkenleri", Description: "Laravel .env dosyalarındaki hassas değerleri bulur.", Example: `site:github.com "{domain}" ".env" "DB_USERNAME" -homestead`, Tags: []string{"laravel", "env"}},
	{ID: "github-smtp-env", Category: "GitHub", Title: "GitHub: SMTP Yapılandırmaları", Description: "Mail sunucu ve şifre yapılandırmalarının sızıntılarını arar.", Example: `site:github.com "{domain}" ".env" "MAIL_HOST=smtp.gmail.com"`, Tags: []string{"smtp", "env"}},
	{ID: "github-git-credentials", Category: "GitHub", Title: "GitHub: Git Credentials", Description: "Düz metin saklanan Git kimlik doğrulama dosyalarını arar.", Example: `site:github.com "{domain}" ".git-credentials"`, Tags: []string{"git", "credentials"}},
	{ID: "github-bash-history", Category: "GitHub", Title: "GitHub: Bash History", Description: "Terminal geçmişi içinde kalmış hassas komutları arar.", Example: `site:github.com "{domain}" ".bash_history"`, Tags: []string{"bash", "history"}},
	{ID: "github-netrc", Category: "GitHub", Title: "GitHub: .netrc Kimlik Bilgileri", Description: "Otomasyon şifrelerini barındıran .netrc dosyalarını hedefler.", Example: `site:github.com "{domain}" ".netrc" "password"`, Tags: []string{"netrc", "password"}},
	{ID: "github-oauth", Category: "GitHub", Title: "GitHub: OAuth Jetonları", Description: "Hub config içindeki OAuth yetkilendirme jetonlarını arar.", Example: `site:github.com "{domain}" "oauth_token"`, Tags: []string{"oauth", "token"}},
	{ID: "github-robomongo", Category: "GitHub", Title: "GitHub: MongoDB (Robomongo)", Description: "Robomongo istemcisine ait MongoDB kimlik dosyalarını arar.", Example: `site:github.com "{domain}" "robomongo.json"`, Tags: []string{"mongodb", "json"}},
	{ID: "github-filezilla", Category: "GitHub", Title: "GitHub: FileZilla FTP Şifreleri", Description: "FileZilla profilindeki FTP parolalarını arar.", Example: `site:github.com "{domain}" "filezilla.xml" "Pass"`, Tags: []string{"ftp", "password"}},
	{ID: "github-db-connections", Category: "GitHub", Title: "GitHub: Veritabanı Connection XML", Description: "DB connection string içeren XML dosyalarını arar.", Example: `site:github.com "{domain}" "connections.xml"`, Tags: []string{"database", "xml"}},
	{ID: "github-pgpass", Category: "GitHub", Title: "GitHub: PostgreSQL Şifreleri", Description: "PostgreSQL .pgpass dosyası sızıntılarını arar.", Example: `site:github.com "{domain}" ".pgpass"`, Tags: []string{"postgres", "password"}},
	{ID: "github-proftpd", Category: "GitHub", Title: "GitHub: ProFTPD Şifreleri", Description: "cPanel ve ProFTPD yapılandırmalarındaki şifre dosyalarını arar.", Example: `site:github.com "{domain}" "proftpdpasswd"`, Tags: []string{"proftpd", "password"}},
	{ID: "github-sshd-config", Category: "GitHub", Title: "GitHub: SSHD Config", Description: "OpenSSH sunucu yapılandırması dosyalarını arar.", Example: `site:github.com "{domain}" "sshd_config"`, Tags: []string{"sshd", "config"}},
	{ID: "github-dhcp", Category: "GitHub", Title: "GitHub: DHCP Config", Description: "İç ağ yapısını sızdırabilecek DHCP servis yapılandırmalarını arar.", Example: `site:github.com "{domain}" "dhcpd.conf"`, Tags: []string{"dhcp", "network"}},
	{ID: "github-joomla", Category: "GitHub", Title: "GitHub: Joomla DB Passwords", Description: "Joomla konfigürasyonlarındaki veritabanı parolalarını arar.", Example: `site:github.com "{domain}" "configuration.php" "JConfig" "password"`, Tags: []string{"joomla", "password"}},
	{ID: "github-shodan", Category: "GitHub", Title: "GitHub: Shodan API Key", Description: "Python betikleri içinde unutulmuş Shodan API anahtarlarını arar.", Example: `site:github.com "{domain}" "shodan_api_key"`, Tags: []string{"shodan", "api-key"}},
	{ID: "github-shadow", Category: "GitHub", Title: "GitHub: Shadow Dosyaları", Description: "Unix/Linux shadow parola hash dosyalarının GitHub sızıntılarını arar.", Example: `site:github.com "{domain}" "/etc/shadow"`, Tags: []string{"shadow", "hash"}},
	{ID: "github-stripe", Category: "GitHub", Title: "GitHub: Stripe Secret Key", Description: "Stripe ödeme altyapısına ait live anahtarları arar.", Example: `site:github.com "{domain}" "sk_live_" OR "pk_live_"`, Tags: []string{"stripe", "secret"}},
	{ID: "github-openai", Category: "GitHub", Title: "GitHub: OpenAI API Key", Description: "OpenAI veya LLM servisi anahtarı izlerini arar.", Example: `site:github.com "{domain}" "sk-" "org-"`, Tags: []string{"openai", "api-key"}},
	{ID: "github-slack-webhook", Category: "GitHub", Title: "GitHub: Slack Webhook", Description: "Slack mesaj entegrasyonu webhook URL ifşalarını arar.", Example: `site:github.com "{domain}" "https://hooks.slack.com/services/"`, Tags: []string{"slack", "webhook"}},
	{ID: "github-jwt", Category: "GitHub", Title: "GitHub: JWT Secret", Description: "JWT veya secret içeren kod ve yapılandırma izlerini arar.", Example: `site:github.com "{domain}" "eyJ" "secret"`, Tags: []string{"jwt", "secret"}},
	{ID: "multi-password-files", Category: "Kimlik Bilgileri", Title: "Kritik Dosya ve Parola Avı", Description: "SQL, BAK, ENV ve LOG uzantılarında parola izleri arar.", Example: `ext:sql | ext:bak | ext:env | ext:log intext:"password"`, Tags: []string{"password", "multi-file"}},
	{ID: "debug-logs", Category: "Hata ve Log", Title: "Hata Ayıklama Logları", Description: "Log veya metin dosyalarındaki kritik sistem hatalarını arar.", Example: `ext:log | ext:txt intext:"Stack trace" | intext:"Fatal error"`, Tags: []string{"debug", "log"}},
	{ID: "sqlite-leaks", Category: "Veritabanı", Title: "SQLite Veritabanı Sızıntıları", Description: "Açıkta kalmış SQLite veritabanı dosyalarını hedefler.", Example: `ext:sqlite | ext:db | ext:db3 | ext:sqlite3 intext:"CREATE TABLE"`, Tags: []string{"sqlite", "database"}},
	{ID: "config-api-keys", Category: "Kimlik Bilgileri", Title: "Genel Yapılandırma Dosyaları", Description: "CFG, CONF veya CONFIG dosyalarında API anahtarı arar.", Example: `ext:config | ext:conf | ext:cfg intext:"api_key" | intext:"apikey"`, Tags: []string{"config", "api-key"}},
	{ID: "phpinfo", Category: "Teknoloji İfşası", Title: "Genişletilmiş PHP Bilgisi", Description: "PHP dosyalarında unutulmuş phpinfo() çıktılarını hedefler.", Example: `ext:php intitle:"phpinfo()" "System Root"`, Tags: []string{"php", "phpinfo"}},
	{ID: "terminal-history", Category: "Kimlik Bilgileri", Title: "Sunucu Terminal Geçmişi", Description: "Komut geçmişlerinde kalan kimlik bilgisi izlerini arar.", Example: `ext:txt | ext:history intext:"mysql -u" | intext:"curl -H \"Authorization:"`, Tags: []string{"history", "terminal"}},
	{ID: "sensitive-spreadsheets", Category: "Belge", Title: "Hassas Tablo Verileri", Description: "Çalışan veya müşteri verisi barındırabilecek elektronik tabloları hedefler.", Example: `ext:csv | ext:xlsx | ext:xls intext:"email" intext:"password" | intext:"dob"`, Tags: []string{"csv", "xlsx"}},
	{ID: "container-secrets", Category: "Konteyner", Title: "Docker ve Konteyner Şifreleri", Description: "YML/YAML dosyalarındaki veritabanı veya container şifrelerini arar.", Example: `ext:yml | ext:yaml intext:"MYSQL_ROOT_PASSWORD" | intext:"POSTGRES_PASSWORD"`, Tags: []string{"docker", "yaml"}},
	{ID: "asymmetric-private-keys", Category: "Kimlik Bilgileri", Title: "Açık RSA/SSH Özel Anahtarları", Description: "PEM, KEY veya TXT olarak sızdırılmış özel anahtarları arar.", Example: `ext:pem | ext:key | ext:txt "-----BEGIN RSA PRIVATE KEY-----"`, Tags: []string{"rsa", "ssh"}},
	{ID: "json-oauth", Category: "Kimlik Bilgileri", Title: "JSON Yapılandırma Sızıntıları", Description: "JSON dosyalarındaki OAuth veya yetkilendirme anahtarlarını bulur.", Example: `ext:json intext:"client_secret" | intext:"client_id"`, Tags: []string{"json", "oauth"}},
	{ID: "iis-web-config", Category: "Veritabanı", Title: "IIS Web.Config Bağlantı Dizeleri", Description: "Windows/IIS veritabanı bağlantı parolalarını arar.", Example: `ext:config intext:"connectionString" intext:"Password="`, Tags: []string{"iis", "connection-string"}},
	{ID: "vpn-config", Category: "Ağ", Title: "Açık VPN Yapılandırmaları", Description: "OVPN dosyalarındaki ağ ve yetkilendirme yapılandırmalarını arar.", Example: `ext:ovpn intext:"remote" intext:"auth-user-pass"`, Tags: []string{"vpn", "ovpn"}},
	{ID: "spring-secrets", Category: "Java", Title: "Spring Boot / Java Özellikleri", Description: "Spring properties veya yaml dosyalarındaki veritabanı parolalarını arar.", Example: `ext:properties | ext:yaml intext:"spring.datasource.password"`, Tags: []string{"spring", "java"}},
	{ID: "credential-text-files", Category: "Kimlik Bilgileri", Title: "Kimlik Bilgisi Dosyaları", Description: "Adında veya içinde admin şifreleri barındıran düz metin dosyaları arar.", Example: `ext:txt intitle:"passwords" | intitle:"credentials" | intext:"admin:password"`, Tags: []string{"credentials", "txt"}},
	{ID: "registry-exports", Category: "Windows", Title: "Windows Kayıt Defteri Çıktıları", Description: "Dışarı aktarılmış REG dosyalarında parola veya secret izleri arar.", Example: `ext:reg intext:"Password" | intext:"Secret"`, Tags: []string{"windows", "registry"}},
	{ID: "aws-cli-config", Category: "Bulut", Title: "AWS CLI Config Dosyaları", Description: "Amazon Web Services kimlik bilgilerini düz metin olarak arar.", Example: `ext:txt | ext:bak "aws_access_key_id" "aws_secret_access_key"`, Tags: []string{"aws", "cli"}},
	{ID: "php-db-connections", Category: "Veritabanı", Title: "PHP Veritabanı Bağlantıları", Description: "PHP scriptlerinde hardcoded bırakılmış mysql bağlantı verilerini arar.", Example: `ext:php intext:"$db_password" | intext:"mysql_connect"`, Tags: []string{"php", "mysql"}},
	{ID: "temp-code-files", Category: "Kaynak Kod", Title: "Eski ve Geçici Dosyalar", Description: "Editörlerin veya sunucuların bıraktığı eski/geçici kod dosyalarını arar.", Example: `ext:swp | ext:old | ext:tmp | ext:inc intext:"<?php"`, Tags: []string{"tmp", "source"}},
	{ID: "ini-passwords", Category: "Kimlik Bilgileri", Title: "INI Yapılandırma Şifreleri", Description: "INI dosyalarındaki kullanıcı adı ve şifre alanlarını arar.", Example: `ext:ini intext:"pwd" | intext:"password" | intext:"user"`, Tags: []string{"ini", "password"}},
	{ID: "java-keystores", Category: "Sertifika", Title: "Java Keystore ve Sertifikalar", Description: "JKS, keystore veya P12 formatındaki sertifika ve imzalama dosyalarını arar.", Example: `ext:jks | ext:keystore | ext:p12 inurl:config | inurl:ssl`, Tags: []string{"java", "certificate"}},
})

func BuildDorks(domain string) []GeneratedDork {
	results := make([]GeneratedDork, 0, len(DorkLibrary))
	for _, dork := range DorkLibrary {
		results = append(results, BuildDork(domain, dork))
	}
	return results
}

func BuildDork(domain string, dork DorkItem) GeneratedDork {
	query := SiteQuery(domain, dork.Example)
	return GeneratedDork{
		Title:       dork.Title,
		Category:    dork.Category,
		Description: dork.Description,
		Query:       query,
		URL:         GoogleSearchURL(query),
	}
}

func BuildCustomDork(domain, expression string) GeneratedDork {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		expression = DefaultCustomDork
	}

	query := SiteQuery(domain, expression)
	return GeneratedDork{
		Title:       "Özel Sorgu",
		Category:    "Özel",
		Description: "Kullanıcının girdiği özel dork ifadesi hedef domain ile birleştirildi.",
		Query:       query,
		URL:         GoogleSearchURL(query),
	}
}

func QueryStrings(results []GeneratedDork) []string {
	queries := make([]string, 0, len(results))
	for _, result := range results {
		queries = append(queries, result.Query)
	}
	return queries
}

func SiteQuery(domain, expression string) string {
	domain = strings.TrimSpace(domain)
	expression = strings.TrimSpace(expression)
	if strings.Contains(expression, domainToken) {
		return strings.ReplaceAll(expression, domainToken, domain)
	}
	return fmt.Sprintf("site:%s %s", domain, expression)
}

func GoogleSearchURL(query string) string {
	return fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(query))
}

func DorksByCategory(category string) []DorkItem {
	category = strings.TrimSpace(category)
	if category == "" {
		return append([]DorkItem(nil), DorkLibrary...)
	}

	var filtered []DorkItem
	for _, dork := range DorkLibrary {
		if strings.EqualFold(dork.Category, category) {
			filtered = append(filtered, dork)
		}
	}
	return filtered
}

func DorkCategories() []string {
	seen := make(map[string]struct{})
	categories := make([]string, 0)
	for _, dork := range DorkLibrary {
		if _, ok := seen[dork.Category]; ok {
			continue
		}
		seen[dork.Category] = struct{}{}
		categories = append(categories, dork.Category)
	}
	sort.Strings(categories)
	return categories
}

func normalizeDorkLibrary(items []DorkItem) []DorkItem {
	seen := make(map[string]struct{}, len(items))
	normalized := make([]DorkItem, 0, len(items))

	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Category = strings.TrimSpace(item.Category)
		item.Title = strings.TrimSpace(item.Title)
		item.Description = strings.TrimSpace(item.Description)
		item.Example = strings.TrimSpace(item.Example)

		if item.ID == "" || item.Title == "" || item.Example == "" {
			continue
		}

		key := strings.ToLower(item.Example)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		normalized = append(normalized, item)
	}

	return normalized
}
