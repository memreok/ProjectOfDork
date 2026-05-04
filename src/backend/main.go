package main

import (
	"dork-project/database"
	"dork-project/handlers"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		domain := r.FormValue("domain")
		if domain == "" {
			domain = r.URL.Query().Get("domain")
		}

		domainLog := ""
		if domain != "" {
			domainLog = fmt.Sprintf(" | Hedef: %s%s%s", ColorCyan, domain, ColorReset)
		}

		logMessage := fmt.Sprintf("[%s%s%s] %s%s%s %v%s",
			ColorYellow, r.Method, ColorReset,
			ColorBlue, r.URL.Path, ColorReset,
			duration,
			domainLog,
		)
		fmt.Println(logMessage)
	}
}

func main() {
	godotenv.Load()

	database.InitDB()

	http.HandleFunc("/", loggerMiddleware(handlers.FormHandler))
	http.HandleFunc("/api/dorks", loggerMiddleware(handlers.ApiHandler))

	port := ":9867"
	fmt.Println("=====================================================")
	fmt.Printf(" %s[BAŞLADI]%s Dork Atölyesi v1.5 (PostgreSQL Aktif)\n", ColorGreen, ColorReset)
	fmt.Printf(" [WEB] Arayüz: http://localhost%s\n", port)
	fmt.Printf(" [API] Endpoint: http://localhost%s/api/dorks?domain=ornek.com\n", port)
	fmt.Println("=====================================================")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
