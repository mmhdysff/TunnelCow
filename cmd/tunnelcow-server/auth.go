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
    <title>TunnelCow Login</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-page: #000000;
            --bg-card: rgba(24, 24, 27, 0.4);
            --border: #27272a;
            --input-bg: rgba(0, 0, 0, 0.6);
            --primary: #ffffff;
            --text-main: #f4f4f5;
            --text-sub: #a1a1aa;
            --danger: #ef4444;
            --danger-bg: rgba(239, 68, 68, 0.15);
        }
        
        body {
            margin: 0;
            padding: 0;
            background-color: var(--bg-page);
            font-family: 'Inter', sans-serif;
            color: var(--text-main);
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            background-image: 
                radial-gradient(circle at 50%% 0%%, #18181b 0%%, transparent 70%%),
                linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
            background-size: 100%% 100%%, 50px 50px, 50px 50px;
        }

        .login-wrapper {
            width: 100%%;
            max-width: 380px;
            padding: 24px;
        }

        .card {
            background: var(--bg-card);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid var(--border);
            border-radius: 16px;
            padding: 40px;
            box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.05), 0 20px 40px -10px rgba(0, 0, 0, 0.5);
            text-align: center;
            position: relative;
            overflow: hidden;
        }

        .card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 1px;
            background: linear-gradient(90deg, transparent, rgba(255,255,255,0.2), transparent);
        }

        .logo-area {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 56px;
            height: 56px;
            background: linear-gradient(135deg, #3f3f46, #18181b);
            border-radius: 12px;
            margin-bottom: 24px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            border: 1px solid rgba(255,255,255,0.1);
        }

        .logo-icon {
            color: white;
        }

        h1 {
            font-size: 20px;
            font-weight: 700;
            letter-spacing: -0.025em;
            margin: 0 0 8px 0;
            color: white;
        }

        p.subtitle {
            font-size: 13px;
            color: var(--text-sub);
            margin: 0 0 32px 0;
            line-height: 1.5;
        }

        .domain-tag {
            display: inline-block;
            background: #27272a;
            color: #e4e4e7;
            padding: 4px 10px;
            border-radius: 6px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            margin-top: 4px;
            border: 1px solid rgba(255,255,255,0.1);
        }

        .form-group {
            margin-bottom: 16px;
            text-align: left;
        }

        label {
            display: block;
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-sub);
            margin-bottom: 8px;
        }

        input {
            width: 100%%;
            padding: 12px 14px;
            background: var(--input-bg);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: white;
            font-size: 14px;
            font-family: 'Inter', sans-serif;
            box-sizing: border-box;
            transition: all 0.2s;
        }

        input:focus {
            outline: none;
            border-color: #52525b;
            background: rgba(0, 0, 0, 0.8);
            box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.05);
        }

        button {
            width: 100%%;
            padding: 12px;
            background: white;
            color: black;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            margin-top: 12px;
        }

        button:hover {
            background: #e4e4e7;
            transform: translateY(-1px);
        }

        button:active {
            transform: translateY(0);
        }

        .error-msg {
            background: var(--danger-bg);
            border: 1px solid rgba(239, 68, 68, 0.3);
            color: #fca5a5;
            font-size: 13px;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 24px;
            text-align: left;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .footer {
            margin-top: 32px;
            font-size: 12px;
            color: #52525b;
        }
    </style>
</head>
<body>
    <div class="login-wrapper">
        <div class="card">
            <div class="logo-area">
                <svg class="logo-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
                </svg>
            </div>
            <h1>Restricted Area</h1>
            <p class="subtitle">
                Authentication required for <br>
                <span class="domain-tag">%s</span>
            </p>

            %s

            <form method="POST">
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" name="username" required autocomplete="off" autofocus placeholder="Enter username">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" name="password" required placeholder="Enter password">
                </div>
                <button type="submit">Unlock Access</button>
            </form>
            
            <div class="footer">
                Powered by TunnelCow
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
