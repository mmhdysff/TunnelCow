package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

const loginHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TunnelCow Auth</title>
    <style>
        :root {
            --bg-color: #09090b;
            --card-bg: rgba(24, 24, 27, 0.6);
            --border-color: rgba(63, 63, 70, 0.4);
            --text-color: #e4e4e7;
            --text-muted: #a1a1aa;
            --primary-color: #fafafa;
            --primary-bg: #18181b;
            --error-color: #ef4444;
        }
        body {
            margin: 0;
            padding: 0;
            background-color: var(--bg-color);
            color: var(--text-color);
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            background-image: radial-gradient(circle at center, #18181b 0%%, #000000 100%%);
        }
        .container {
            width: 100%;
            max-width: 400px;
            padding: 20px;
        }
        .card {
            background: var(--card-bg);
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 40px 30px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            text-align: center;
        }
        .logo {
            width: 48px;
            height: 48px;
            background-color: white;
            color: black;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 20px;
            font-weight: 900;
            font-size: 24px;
        }
        h1 {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 8px;
            color: var(--text-color);
        }
        p {
            font-size: 14px;
            color: var(--text-muted);
            margin-bottom: 30px;
            margin-top: 0;
        }
        .form-group {
            margin-bottom: 16px;
            text-align: left;
        }
        label {
            display: block;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--text-muted);
            margin-bottom: 6px;
        }
        input {
            width: 100%;
            padding: 12px;
            background: rgba(0, 0, 0, 0.3);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            color: white;
            font-size: 14px;
            box-sizing: border-box;
            transition: border-color 0.2s;
        }
        input:focus {
            outline: none;
            border-color: var(--text-muted);
        }
        button {
            width: 100%;
            padding: 12px;
            background: var(--primary-color);
            color: black;
            border: none;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: opacity 0.2s;
            margin-top: 10px;
        }
        button:hover {
            opacity: 0.9;
        }
        .error {
            background-color: rgba(239, 68, 68, 0.1);
            border: 1px solid rgba(239, 68, 68, 0.2);
            color: var(--error-color);
            font-size: 13px;
            padding: 10px;
            border-radius: 6px;
            margin-bottom: 20px;
        }
        .footer {
            margin-top: 30px;
            font-size: 12px;
            color: #52525b;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <div class="logo">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="8" x2="12" y2="12"></line>
                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
            </div>
            <h1>Restricted Access</h1>
            <p>Please authenticate to access %s</p>

            %s

            <form method="POST">
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" name="username" required autocomplete="off" autofocus>
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" name="password" required>
                </div>
                <button type="submit">Enter Tunnel</button>
            </form>
            
            <div class="footer">
                Secured by TunnelCow
            </div>
        </div>
    </div>
</body>
</html>
`

func generateCookie(domain, user, pass, secret string) *http.Cookie {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(domain + user + pass))
	token := hex.EncodeToString(h.Sum(nil))

	nameHash := sha256.Sum256([]byte(domain))
	return &http.Cookie{
		Name:     "tc_auth_" + hex.EncodeToString(nameHash[:])[:8],
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7,
	}
}

func validateCookie(r *http.Request, domain, user, pass, secret string) bool {
	nameHash := sha256.Sum256([]byte(domain))
	cookieName := "tc_auth_" + hex.EncodeToString(nameHash[:])[:8]
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(domain + user + pass))
	expectedToken := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(cookie.Value), []byte(expectedToken))
}
