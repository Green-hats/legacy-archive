package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown"

	"ani-rss/internal/model"
)

var mailHTMLTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8"/>
<title>ANI-RSS</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
</head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f5f6fa;margin:0;padding:24px;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,.1);overflow:hidden;">
{{IMAGE}}
<div style="padding:24px;color:#333;line-height:1.6;">
{{RENDER}}
</div>
</div>
</body>
</html>`

// SendMail sends an HTML email via SMTP (mirrors MailNotification.send).
func SendMail(cfg *model.NotificationConfig, ani *model.Ani, text string, status model.NotificationStatusEnum) error {
	if strings.TrimSpace(cfg.MailFrom) == "" || strings.TrimSpace(cfg.MailSMTPHost) == "" ||
		strings.TrimSpace(cfg.MailPassword) == "" || strings.TrimSpace(cfg.MailAddressee) == "" {
		return fmt.Errorf("邮件配置不完整")
	}
	body := ReplaceTemplate(ani, cfg, text, status)
	body = strings.ReplaceAll(body, "\n", "\n\n")
	render := string(markdown.ToHTML([]byte(body), nil, nil))

	imageBlock := ""
	if cfg.MailImage && ani != nil && ani.Image != "" {
		imageBlock = fmt.Sprintf(`<img src="%s" alt="cover" style="width:100%%;max-height:320px;object-fit:cover;"/>`, ani.Image)
	}
	html := strings.ReplaceAll(mailHTMLTemplate, "{{IMAGE}}", imageBlock)
	html = strings.ReplaceAll(html, "{{RENDER}}", render)

	title := text
	if len(text) > 200 && ani != nil {
		title = ani.Title
	}
	return sendSMTP(cfg, title, html)
}

func sendSMTP(cfg *model.NotificationConfig, subject, htmlBody string) error {
	host := cfg.MailSMTPHost
	port := cfg.MailSMTPPort
	if port == 0 {
		port = 465
	}
	from := cfg.MailFrom
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	// Sender and recipients
	sender := fmt.Sprintf("ani-rss <%s>", from)

	var auth smtp.Auth
	// Java always authenticates with the configured credentials
	auth = smtp.PlainAuth("", from, cfg.MailPassword, host)

	msg := buildMimeMessage(sender, cfg.MailAddressee, subject, htmlBody)

	if cfg.MailSSLEnable {
		// implicit TLS
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return err
		}
		defer client.Close()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
		if err := client.Mail(from); err != nil {
			return err
		}
		if err := client.Rcpt(cfg.MailAddressee); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return err
		}
		return w.Close()
	}

	// plain or STARTTLS
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if cfg.MailTLSEnable {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(cfg.MailAddressee); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

func buildMimeMessage(from, to, subject, htmlBody string) string {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + encodeSubject(subject) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(htmlBody)
	return sb.String()
}

func encodeSubject(s string) string {
	// keep it simple: raw UTF-8 subject works for most clients
	return s
}