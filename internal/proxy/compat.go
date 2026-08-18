package proxy

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"
)

// CheckPortConflict checks if a port is already in use by another process.
func CheckPortConflict(port int) (string, bool) {
	// Check if something is listening
	addr := fmt.Sprintf(":%d", port)
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		conn.Close()
		return "", true // port in use but can't identify process
	}

	// Try to identify what's using the port (Linux)
	out, _ := exec.Command("ss", "-tlnp", "sport", "=", fmt.Sprintf(":%d", port)).CombinedOutput()
	if len(out) > 0 {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines[1:] { // skip header
			if strings.TrimSpace(line) != "" {
				return strings.TrimSpace(line), true
			}
		}
	}
	return "", false
}

// DetectNginx checks if nginx is installed and running.
func DetectNginx() bool {
	out, _ := exec.Command("which", "nginx").CombinedOutput()
	return len(out) > 0
}

// GenerateNginxSnippet returns a recommended nginx config snippet
// for proxying to WAF backend.
func GenerateNginxSnippet(wafAddr, backendAddr string) string {
	return fmt.Sprintf(`# ZhiYu-WAF reverse proxy configuration
# Place this in your nginx server block

upstream zhiyu_waf {
    server %s;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://zhiyu_waf;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# V2 manages temporary malicious-IP blocks through its dedicated nftables table.
# Keep normal ingress routing under your existing proxy or load balancer.
`, wafAddr)
}

// LogCompatAdvice logs compatibility recommendations at startup.
func LogCompatAdvice(proxyPort, backendPort int) {
	if DetectNginx() {
		log.Println("=== Nginx detected ===")
		log.Println("V2 deployment: use nginx as the frontend and proxy protected traffic to the WAF listen port")
		log.Printf("  Sample nginx config: proxy_pass http://127.0.0.1:%d;", proxyPort)

		// Check for common conflicts
		for _, port := range []int{80, 443} {
			if info, found := CheckPortConflict(port); found {
				log.Printf("  WARNING: port %d is in use: %s", port, info)
			}
		}
	}
}
