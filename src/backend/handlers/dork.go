package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"dork-project/database"
	"dork-project/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	historySessionCookie = "dork_session"
	adminSessionCookie   = "dork_admin"
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
	sessionID := historySessionID(w, r)

	if database.IsReady() {
		database.DB.Where("session_id = ?", sessionID).Order("timestamp desc").Limit(5).Find(&data.History)
	}

	if r.Method == http.MethodPost {
		if !allowRequest(r, "form", 30, time.Minute) {
			data.ErrorMessage = "Kısa sürede çok fazla istek gönderildi. Lütfen biraz sonra tekrar dene."
			tmpl.Execute(w, data)
			return
		}

		actionType := r.FormValue("action")
		adminScope := r.FormValue("history_scope") == "all" && canManageHistory(r)

		if actionType == "clear_history" {
			if database.IsReady() {
				if adminScope {
					database.DB.Where("1 = 1").Delete(&models.HistoryItem{})
				} else {
					database.DB.Where("session_id = ?", sessionID).Delete(&models.HistoryItem{})
				}
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
			historyID := r.FormValue("history_id")
			if database.IsReady() {
				if adminScope {
					database.DB.Delete(&models.HistoryItem{}, historyID)
				} else {
					database.DB.Where("id = ? AND session_id = ?", historyID, sessionID).Delete(&models.HistoryItem{})
				}
				loadHistory(&data, sessionID, adminScope)
			}
			if shouldReturnToHistory(r) {
				http.Redirect(w, r, "/history", http.StatusSeeOther)
				return
			}
			tmpl.Execute(w, data)
			return
		}

		if actionType == "cleanup_history" {
			if database.IsReady() {
				cleanupDuplicateHistory(sessionID, adminScope)
				loadHistory(&data, sessionID, adminScope)
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
			database.DB.Where("id = ? AND session_id = ?", historyID, sessionID).First(&historyRecord)
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
			result := database.DB.Where("session_id = ? AND LOWER(domain) = ?", sessionID, hedefDomain).First(&existing)
			if result.Error != nil {
				database.DB.Create(&models.HistoryItem{
					SessionID:       sessionID,
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
			database.DB.Where("session_id = ?", sessionID).Order("timestamp desc").Find(&data.History)
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
	if providedToken != "" {
		return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
	}

	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return false
	}
	expectedSession := adminSessionSignature(expectedToken)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expectedSession)) == 1
}

func setAdminSession(w http.ResponseWriter) bool {
	adminToken := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	if adminToken == "" {
		return false
	}

	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    adminSessionSignature(adminToken),
		Path:     "/",
		MaxAge:   60 * 60 * 8,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return true
}

func clearAdminSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func adminSessionSignature(adminToken string) string {
	mac := hmac.New(sha256.New, []byte(adminToken))
	mac.Write([]byte("dork-admin-session"))
	return hex.EncodeToString(mac.Sum(nil))
}

func shouldReturnToHistory(r *http.Request) bool {
	return r.FormValue("return_to") == "/history"
}

func historySessionID(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(historySessionCookie); err == nil {
		sessionID := strings.TrimSpace(cookie.Value)
		if len(sessionID) >= 32 {
			return sessionID
		}
	}

	sessionID := newSessionID()
	http.SetCookie(w, &http.Cookie{
		Name:     historySessionCookie,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 180,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return sessionID
}

func newSessionID() string {
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(token)
}

func loadHistory(data *models.PageData, sessionID string, adminScope bool) {
	query := database.DB.Order("timestamp desc")
	if !adminScope {
		query = query.Where("session_id = ?", sessionID)
	}
	query.Find(&data.History)
}

func cleanupDuplicateHistory(sessionID string, adminScope bool) {
	var history []models.HistoryItem
	query := database.DB.Order("timestamp desc")
	if !adminScope {
		query = query.Where("session_id = ?", sessionID)
	}
	query.Find(&history)

	seen := make(map[string]models.HistoryItem)
	for _, item := range history {
		normalizedDomain := normalizeTarget(item.Domain)
		if normalizedDomain == "" {
			continue
		}

		key := item.SessionID + "|" + normalizedDomain
		if kept, ok := seen[key]; ok {
			database.DB.Delete(&models.HistoryItem{}, item.ID)
			if kept.Domain != normalizedDomain {
				kept.Domain = normalizedDomain
				database.DB.Save(&kept)
				seen[key] = kept
			}
			continue
		}

		if item.Domain != normalizedDomain {
			item.Domain = normalizedDomain
			database.DB.Save(&item)
		}
		seen[key] = item
	}
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("../frontend/history.html"))
	sessionID := historySessionID(w, r)

	if r.Method == http.MethodPost {
		switch r.FormValue("action") {
		case "admin_login":
			if canManageHistory(r) {
				setAdminSession(w)
			}
			http.Redirect(w, r, "/history", http.StatusSeeOther)
			return
		case "admin_logout":
			clearAdminSession(w)
			http.Redirect(w, r, "/history", http.StatusSeeOther)
			return
		}
	}

	var history []models.HistoryItem
	isAdmin := canManageHistory(r)
	if database.IsReady() {
		query := database.DB.Order("timestamp desc")
		if !isAdmin {
			query = query.Where("session_id = ?", sessionID)
		}
		query.Find(&history)
	}

	tmpl.Execute(w, struct {
		History []models.HistoryItem
		IsAdmin bool
	}{History: history, IsAdmin: isAdmin})
}
