package handlers

import (
	"crypto/subtle"
	"dork-project/database"
	"dork-project/models"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	domainRegex         = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
	wildcardDomainRegex = regexp.MustCompile(`^\*\.(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z]{2,}$`)
)

func FormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../frontend/index.html"))

	data := models.PageData{
		Dorks:      models.DorkLibrary,
		Categories: models.DorkCategories(),
	}

	if database.IsReady() {
		database.DB.Order("timestamp desc").Limit(5).Find(&data.History)
	}

	if r.Method == http.MethodPost {
		if !allowRequest(r, "form", 30, time.Minute) {
			data.ErrorMessage = "Kısa sürede çok fazla istek gönderildi. Lütfen biraz sonra tekrar dene."
			tmpl.Execute(w, data)
			return
		}

		actionType := r.FormValue("action")

		if actionType == "clear_history" {
			if !canManageHistory(r) {
				data.ErrorMessage = "Geçmişi silmek için admin yetkisi gerekli."
				tmpl.Execute(w, data)
				return
			}
			if database.IsReady() {
				database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.HistoryItem{})
			}
			data.History = []models.HistoryItem{}
			if shouldReturnToHistory(r) {
				http.Redirect(w, r, "/history", http.StatusSeeOther)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		if actionType == "delete_history" {
			if !canManageHistory(r) {
				data.ErrorMessage = "Geçmiş kaydı silmek için admin yetkisi gerekli."
				tmpl.Execute(w, data)
				return
			}
			historyID := r.FormValue("history_id")
			if database.IsReady() {
				database.DB.Delete(&models.HistoryItem{}, historyID)
				database.DB.Order("timestamp desc").Find(&data.History)
			}
			if shouldReturnToHistory(r) {
				http.Redirect(w, r, "/history", http.StatusSeeOther)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		if actionType == "cleanup_history" {
			if !canManageHistory(r) {
				data.ErrorMessage = "Geçmişi düzenlemek için admin yetkisi gerekli."
				tmpl.Execute(w, data)
				return
			}
			if database.IsReady() {
				cleanupDuplicateHistory()
				database.DB.Order("timestamp desc").Find(&data.History)
			}
			if shouldReturnToHistory(r) {
				http.Redirect(w, r, "/history", http.StatusSeeOther)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		hedefDomain := normalizeTarget(r.FormValue("domain"))
		historyID := r.FormValue("history_id")

		var historyRecord models.HistoryItem
		useStoredData := false

		if actionType == "load_history" && historyID != "" && database.IsReady() {
			database.DB.First(&historyRecord, historyID)
			if historyRecord.ID != 0 {
				hedefDomain = normalizeTarget(historyRecord.Domain)
				data.IsTargetAlive = historyRecord.IsTargetAlive
				data.TargetStatusMsg = historyRecord.TargetStatusMsg
				useStoredData = true
			}
		}

		data.Domain = hedefDomain
		data.SelectedDork = r.FormValue("custom_dork")

		if !isValidTarget(hedefDomain) {
			data.ErrorMessage = "Geçersiz format! Lütfen example.com veya *.example gibi geçerli bir hedef girin."
			tmpl.Execute(w, data)
			return
		}

		if isWildcardTarget(hedefDomain) {
			data.IsTargetAlive = false
			data.TargetStatusMsg = "Wildcard hedef (canlılık kontrolü atlandı)"
		} else if !useStoredData {
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

		}

		if !useStoredData && database.IsReady() {
			var existing models.HistoryItem
			result := database.DB.Where("LOWER(domain) = ?", hedefDomain).First(&existing)
			if result.Error != nil {
				database.DB.Create(&models.HistoryItem{
					Domain:          hedefDomain,
					IsTargetAlive:   data.IsTargetAlive,
					TargetStatusMsg: data.TargetStatusMsg,
				})
			} else {
				existing.Domain = hedefDomain
				existing.IsTargetAlive = data.IsTargetAlive
				existing.TargetStatusMsg = data.TargetStatusMsg
				existing.Timestamp = time.Now()
				database.DB.Save(&existing)
			}
			database.DB.Order("timestamp desc").Find(&data.History)
		}

		secilenDork := data.SelectedDork
		if actionType == "" || actionType == "all" || actionType == "export_txt" || actionType == "export_json" || actionType == "load_history" {
			data.Results = models.BuildDorks(hedefDomain)
		} else {
			data.Results = append(data.Results, models.BuildCustomDork(hedefDomain, secilenDork))
		}
		data.DorkList = models.QueryStrings(data.Results)

		if actionType == "export_txt" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", hedefDomain))
			w.Header().Set("Content-Type", "text/plain")
			for _, res := range data.Results {
				fmt.Fprintf(w, "[%s] %s\nAçıklama: %s\nSorgu: %s\nLink: %s\n\n", res.Category, res.Title, res.Description, res.Query, res.URL)
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

func normalizeTarget(raw string) string {
	target := strings.ToLower(strings.TrimSpace(raw))
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "//")

	if cut := strings.IndexAny(target, "/?#"); cut >= 0 {
		target = target[:cut]
	}

	return strings.Trim(target, ".")
}

func isValidTarget(target string) bool {
	return domainRegex.MatchString(target) || wildcardDomainRegex.MatchString(target)
}

func isWildcardTarget(target string) bool {
	return wildcardDomainRegex.MatchString(target)
}

func canManageHistory(r *http.Request) bool {
	expectedToken := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	if expectedToken == "" {
		return false
	}

	providedToken := strings.TrimSpace(r.FormValue("admin_token"))
	if providedToken == "" {
		providedToken = strings.TrimSpace(r.Header.Get("X-Admin-Token"))
	}

	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
}

func shouldReturnToHistory(r *http.Request) bool {
	return r.FormValue("return_to") == "/history"
}

func cleanupDuplicateHistory() {
	var history []models.HistoryItem
	database.DB.Order("timestamp desc").Find(&history)

	seen := make(map[string]models.HistoryItem)
	for _, item := range history {
		normalizedDomain := normalizeTarget(item.Domain)
		if normalizedDomain == "" {
			continue
		}

		if kept, ok := seen[normalizedDomain]; ok {
			database.DB.Delete(&models.HistoryItem{}, item.ID)
			if kept.Domain != normalizedDomain {
				kept.Domain = normalizedDomain
				database.DB.Save(&kept)
				seen[normalizedDomain] = kept
			}
			continue
		}

		if item.Domain != normalizedDomain {
			item.Domain = normalizedDomain
			database.DB.Save(&item)
		}
		seen[normalizedDomain] = item
	}
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../frontend/history.html"))

	var history []models.HistoryItem
	if database.IsReady() {
		database.DB.Order("timestamp desc").Find(&history)
	}

	tmpl.Execute(w, struct{ History []models.HistoryItem }{History: history})
}
