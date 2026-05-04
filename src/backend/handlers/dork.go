package handlers

import (
	"dork-project/models"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

func FormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../frontend/index.html"))

	data := models.PageData{
		Dorks: models.DorkLibrary,
	}

	if r.Method == http.MethodPost {
		hedefDomain := strings.TrimSpace(r.FormValue("domain"))
		secilenDork := strings.TrimSpace(r.FormValue("custom_dork"))
		actionType := r.FormValue("action")

		data.Domain = hedefDomain
		data.SelectedDork = secilenDork

		if !domainRegex.MatchString(hedefDomain) {
			data.ErrorMessage = "Geçersiz format! Lütfen ornek.com gibi geçerli bir domain girin."
			tmpl.Execute(w, data)
			return
		}

		client := &http.Client{
			Timeout: 5 * time.Second, // 5 saniye bekleme süresi
		}

		makeReq := func(targetURL string) (*http.Response, error) {
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				return nil, err
			}
			// Gerçek bir tarayıcı (Chrome) gibi davranıyoruz
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			return client.Do(req)
		}

		resp, err := makeReq("https://" + hedefDomain)

		if err != nil {
			resp, err = makeReq("http://" + hedefDomain)
		}

		if err != nil {
			data.IsTargetAlive = false

			errString := strings.ToLower(err.Error())
			if strings.Contains(errString, "no such host") {
				data.TargetStatusMsg = "Ulaşılamaz (DNS Hatası/Domain Yok)"
			} else if strings.Contains(errString, "timeout") || strings.Contains(errString, "deadline exceeded") {
				data.TargetStatusMsg = "Ulaşılamaz (Zaman Aşımı - 5s)"
			} else if strings.Contains(errString, "connection refused") {
				data.TargetStatusMsg = "Ulaşılamaz (Bağlantı Reddedildi)"
			} else {
				data.TargetStatusMsg = "Ulaşılamaz (Sunucu Çökmüş)"
			}
		} else {
			defer resp.Body.Close()

			protokol := "HTTP"
			if resp.Request.URL.Scheme == "https" || resp.TLS != nil {
				protokol = "HTTPS"
			}

			data.IsTargetAlive = true

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				data.TargetStatusMsg = fmt.Sprintf("Aktif (%s %d)", protokol, resp.StatusCode)
			} else if resp.StatusCode == 403 || resp.StatusCode == 401 {
				data.TargetStatusMsg = fmt.Sprintf("Korumalı WAF/Auth (%s %d)", protokol, resp.StatusCode)
			} else {
				data.TargetStatusMsg = fmt.Sprintf("Sunucu Aktif ama Hatalı (%s %d)", protokol, resp.StatusCode)
			}
		}
		if actionType == "all" || actionType == "export_txt" || actionType == "export_json" {
			for _, dork := range models.DorkLibrary {
				rawQuery := fmt.Sprintf("site:%s %s", hedefDomain, dork.Example)
				data.Results = append(data.Results, models.GeneratedDork{
					Title: dork.Title,
					Query: rawQuery,
					URL:   fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(rawQuery)),
				})
			}
		} else {
			if secilenDork == "" {
				secilenDork = "ext:sql | ext:env | ext:log"
			}
			rawQuery := fmt.Sprintf("site:%s %s", hedefDomain, secilenDork)
			data.Results = append(data.Results, models.GeneratedDork{
				Title: "Özel Sorgu",
				Query: rawQuery,
				URL:   fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(rawQuery)),
			})
		}

		if actionType == "export_txt" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_dork_raporu.txt", hedefDomain))
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")

			fmt.Fprintf(w, "Hedef: %s için OSINT Dork Raporu\n", hedefDomain)
			fmt.Fprintf(w, strings.Repeat("-", 50)+"\n\n")
			for _, res := range data.Results {
				fmt.Fprintf(w, "[%s]\nSorgu: %s\nLink: %s\n\n", res.Title, res.Query, res.URL)
			}
			return
		}

		if actionType == "export_json" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_dork_raporu.json", hedefDomain))
			w.Header().Set("Content-Type", "application/json")

			json.NewEncoder(w).Encode(data.Results)
			return
		}
	}

	tmpl.Execute(w, data)
}
