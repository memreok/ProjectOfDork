package handlers

import (
	"dork-project/models"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func ApiHandler(w http.ResponseWriter, r *http.Request) {
	// Gelen cevabın bir HTML değil, JSON olduğunu belirtiyoruz
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Sadece GET metodu desteklenir."})
		return
	}

	if !allowRequest(r, "api", 60, time.Minute) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Kısa sürede çok fazla istek gönderildi. Lütfen biraz sonra tekrar deneyin."})
		return
	}

	hedefDomain := normalizeTarget(r.URL.Query().Get("domain"))

	if hedefDomain == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Lütfen 'domain' parametresi gönderin."})
		return
	}

	if !isValidTarget(hedefDomain) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Geçersiz hedef formatı. example.com veya *.example kullanabilirsiniz."})
		return
	}

	isAlive := false
	statusMsg := "Ulaşılamaz"

	if isWildcardTarget(hedefDomain) {
		statusMsg = "Wildcard hedef (canlılık kontrolü atlandı)"
	} else {
		client := &http.Client{Timeout: 5 * time.Second}
		makeReq := func(targetURL string) (*http.Response, error) {
			req, _ := http.NewRequest("GET", targetURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
			return client.Do(req)
		}

		resp, err := makeReq("https://" + hedefDomain)
		if err != nil {
			resp, err = makeReq("http://" + hedefDomain)
		}

		if err == nil {
			defer resp.Body.Close()
			isAlive = true
			protokol := "HTTP"
			if resp.Request.URL.Scheme == "https" || resp.TLS != nil {
				protokol = "HTTPS"
			}
			statusMsg = fmt.Sprintf("Aktif (%s %d)", protokol, resp.StatusCode)
		}
	}
	results := models.BuildDorks(hedefDomain)

	response := map[string]interface{}{
		"target": map[string]interface{}{
			"domain":   hedefDomain,
			"is_alive": isAlive,
			"status":   statusMsg,
		},
		"total_dorks": len(results),
		"dorks":       results,
	}

	json.NewEncoder(w).Encode(response)
}
