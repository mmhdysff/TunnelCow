package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
)

const shieldHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Check - TunnelCow</title>
    <style>
        :root {
            --bg: #09090b;
            --text: #e4e4e7;
            --sub: #71717a;
            --accent: #f59e0b; /* Amber-500 */
        }
        body {
            margin: 0;
            padding: 0;
            background: var(--bg);
            color: var(--text);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100vh;
        }
        .container {
            text-align: center;
            max-width: 400px;
            padding: 20px;
        }
        .spinner {
            width: 24px;
            height: 24px;
            border: 2px solid var(--sub);
            border-top-color: var(--accent);
            border-radius: 50%;
            margin: 0 auto 20px;
            animation: spin 0.8s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        h1 {
            font-size: 16px;
            font-weight: 600;
            margin: 0 0 10px;
            letter-spacing: -0.01em;
        }
        p {
            font-size: 14px;
            color: var(--sub);
            margin: 0;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="spinner"></div>
        <h1>Checking your browser</h1>
        <p id="status">Please wait a moment while we verify your request.</p>
    </div>

    <script>
        const statusEl = document.getElementById('status');
        
        
        setTimeout(() => {
            fetch('/tunnelcow', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded'
                },
                body: 'domain=' + encodeURIComponent(window.location.hostname)
            })
            .then(res => {
                if (res.ok) {
                    window.location.reload();
                } else {
                    statusEl.innerText = "Verification failed. Please refresh.";
                    statusEl.style.color = "#ef4444";
                }
            })
            .catch(() => {
                statusEl.innerText = "Connection error. Please refresh.";
                statusEl.style.color = "#ef4444";
            });
        }, 2000);
    </script>
</body>
</html>`

func validateShieldCookie(r *http.Request, secret string) bool {
	cookie, err := r.Cookie("tc_shield")
	if err != nil {
		return false
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(r.RemoteAddr + secret))
	expected := hex.EncodeToString(h.Sum(nil))

	return cookie.Value == expected
}

func serveChallengePage(w http.ResponseWriter) {
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, shieldHTML)
}

func handleShieldVerify(w http.ResponseWriter, r *http.Request, secret string) {

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(r.RemoteAddr + secret))
	signature := hex.EncodeToString(h.Sum(nil))

	http.SetCookie(w, &http.Cookie{
		Name:     "tc_shield",
		Value:    signature,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   3600 * 24,
	})
	w.WriteHeader(http.StatusOK)
}
