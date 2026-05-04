package handlers

import (
	"dork-project/database"
	"dork-project/models"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

func FormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../frontend/index.html"))

	data := models.PageData{
		Dorks: models.DorkLibrary,
	}

	database.DB.Order("timestamp desc").Find(&data.History)

	if r.Method == http.MethodPost {
		actionType := r.FormValue("action")

		if actionType == "clear_history" {
			database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.HistoryItem{})
			data.History = []models.HistoryItem{}
			tmpl.Execute(w, data)
			return
		}

		if actionType == "delete_history" {
			historyID := r.FormValue("history_id")
			database.DB.Delete(&models.HistoryItem{}, historyID)
			database.DB.Order("timestamp desc").Find(&data.History)
			tmpl.Execute(w, data)
			return
		}

		hedefDomain := strings.TrimSpace(r.FormValue("domain"))
		historyID := r.FormValue("history_id")

		var historyRecord models.HistoryItem
		useStoredData := false

		if actionType == "load_history" && historyID != "" {
			database.DB.First(&historyRecord, historyID)
			if historyRecord.ID != 0 {
				hedefDomain = historyRecord.Domain
				data.IsTargetAlive = historyRecord.IsTargetAlive
				data.TargetStatusMsg = historyRecord.TargetStatusMsg
				useStoredData = true
			}
		}

		data.Domain = hedefDomain
		data.SelectedDork = r.FormValue("custom_dork")

		if !domainRegex.MatchString(hedefDomain) {
			data.ErrorMessage = "Geçersiz format! Lütfen ornek.com gibi geçerli bir domain girin."
			tmpl.Execute(w, data)
			return
		}

		if !useStoredData {
			client := &http.Client{Timeout: 5 * time.Second}
			makeReq := func(targetURL string) (*http.Response, error) {
				req, _ := http.NewRequest("GET", targetURL, nil)
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
					data.TargetStatusMsg = "Ulaşılamaz (DNS Hatası)"
				} else {
					data.TargetStatusMsg = "Ulaşılamaz"
				}
			} else {
				defer resp.Body.Close()
				data.IsTargetAlive = true
				protokol := "HTTP"
				if resp.Request.URL.Scheme == "https" || resp.TLS != nil {
					protokol = "HTTPS"
				}
				data.TargetStatusMsg = fmt.Sprintf("Aktif (%s %d)", protokol, resp.StatusCode)
			}

			var existing models.HistoryItem
			result := database.DB.Where("domain = ?", hedefDomain).First(&existing)
			if result.Error != nil {
				database.DB.Create(&models.HistoryItem{
					Domain:          hedefDomain,
					IsTargetAlive:   data.IsTargetAlive,
					TargetStatusMsg: data.TargetStatusMsg,
				})
			} else {
				existing.IsTargetAlive = data.IsTargetAlive
				existing.TargetStatusMsg = data.TargetStatusMsg
				existing.Timestamp = time.Now()
				database.DB.Save(&existing)
			}
			database.DB.Order("timestamp desc").Find(&data.History)
		}

		secilenDork := data.SelectedDork
		if actionType == "all" || actionType == "export_txt" || actionType == "export_json" || actionType == "load_history" {
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
				secilenDork = "ext:sql | ext:env"
			}
			rawQuery := fmt.Sprintf("site:%s %s", hedefDomain, secilenDork)
			data.Results = append(data.Results, models.GeneratedDork{
				Title: "Özel Sorgu",
				Query: rawQuery,
				URL:   fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(rawQuery)),
			})
		}

		if actionType == "export_txt" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", hedefDomain))
			w.Header().Set("Content-Type", "text/plain")
			for _, res := range data.Results {
				fmt.Fprintf(w, "[%s]\nSorgu: %s\nLink: %s\n\n", res.Title, res.Query, res.URL)
			}
			return
		}

		if actionType == "export_json" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", hedefDomain))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data.Results)
			return
		}
	}

	tmpl.Execute(w, data)
}
