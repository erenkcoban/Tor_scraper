package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// ----------------------------------------------------
// 1. Dosya Okuma Modülü (Input Handler)
// ----------------------------------------------------
func readTargetsFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("girdi dosyası açılamadı: %w", err)
	}
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			targets = append(targets, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("dosya okuma hatası: %w", err)
	}

	return targets, nil
}

// ----------------------------------------------------
// 2. Tor Proxy Client + DOĞRU Health Check
// ----------------------------------------------------
func createTorClient(proxyAddr string) (*http.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 dialer oluşturulamadı: %w", err)
	}

	// --- HEALTH CHECK ---
	conn, err := dialer.Dial("tcp", "check.torproject.org:80")
	if err != nil {
		return nil, fmt.Errorf("Tor proxy çalışıyor fakat dış bağlantı başarısız: %w", err)
	}
	conn.Close()
	// --------------------

	transport := &http.Transport{
		DialContext:           dialer.(proxy.ContextDialer).DialContext,
		DisableKeepAlives:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return client, nil
}

// ----------------------------------------------------
// 3. Tarama + Hata Yönetimi + Output Writer
// ----------------------------------------------------
func scrapeTarget(
	client *http.Client,
	targetURL string,
	outputDir string,
	logFile *os.File,
	index int,
) {
	log := func(level, msg string) {
		entry := fmt.Sprintf("[%s] Scanning: %s -> %s\n", level, targetURL, msg)
		fmt.Print(entry)
		logFile.WriteString(entry)
	}

	u, _ := url.Parse(targetURL)
	baseName := strings.ReplaceAll(u.Host, ".", "_")
	timestamp := time.Now().Format("20060102_150405")
	prefix := fmt.Sprintf("%02d_%s_%s", index, baseName, timestamp)

	resp, err := client.Get(targetURL)
	if err != nil {
		filePath := filepath.Join(outputDir, prefix+"_FAILED.txt")
		os.WriteFile(filePath, []byte(err.Error()), 0644)
		log("ERR", "FAILED (Kaydedildi: "+filePath+")")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		filePath := filepath.Join(outputDir, prefix+"_FAILED.txt")
		os.WriteFile(filePath, []byte(fmt.Sprintf("HTTP STATUS: %d", resp.StatusCode)), 0644)
		log("ERR", "FAILED (HTTP "+fmt.Sprint(resp.StatusCode)+")")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		filePath := filepath.Join(outputDir, prefix+"_FAILED.txt")
		os.WriteFile(filePath, []byte("Response body okunamadı"), 0644)
		log("ERR", "FAILED (Body okunamadı)")
		return
	}

	filePath := filepath.Join(outputDir, prefix+".html")
	os.WriteFile(filePath, body, 0644)
	log("INFO", "SUCCESS (Kaydedildi: "+filePath+")")
}

// ----------------------------------------------------
// 4. Main
// ----------------------------------------------------
func main() {
	const torProxyAddr = "127.0.0.1:9150"
	const outputDir = "output"
	const logFileName = "scan_report.log"

	if len(os.Args) < 2 {
		fmt.Println("Kullanım: go run main.go targets.yaml")
		os.Exit(1)
	}

	targetsFile := os.Args[1]

	targets, err := readTargetsFile(targetsFile)
	if err != nil {
		fmt.Println("HATA:", err)
		os.Exit(1)
	}

	client, err := createTorClient(torProxyAddr)
	if err != nil {
		fmt.Println("KRİTİK HATA:", err)
		os.Exit(1)
	}

	os.MkdirAll(outputDir, 0755)

	logFile, err := os.Create(logFileName)
	if err != nil {
		fmt.Println("Log dosyası oluşturulamadı")
		os.Exit(1)
	}
	defer logFile.Close()

	fmt.Printf("--- Tarama Başlatıldı (%d hedef) ---\n", len(targets))
	logFile.WriteString("Tarama başlatıldı: " + time.Now().Format(time.RFC3339) + "\n")

	for i, target := range targets {
		scrapeTarget(client, target, outputDir, logFile, i+1)
	}

	fmt.Println("--- Tarama Tamamlandı ---")
	fmt.Println("Rapor:", logFileName)
	fmt.Println("Çıktılar:", outputDir)
}
