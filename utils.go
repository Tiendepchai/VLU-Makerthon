package utils

import (
    "fmt"
    "log"
    "net/smtp"
    "os"
    "strings"
)

// SendEmail gửi email với định dạng HTML
func SendEmail(to, subject, htmlBody string) error {
    from := os.Getenv("SMTP_FROM")
    password := os.Getenv("SMTP_PASSWORD")
    smtpHost := os.Getenv("SMTP_HOST")
    smtpPort := os.Getenv("SMTP_PORT")
    
    // Tạo header email
    headers := make(map[string]string)
    headers["From"] = from
    headers["To"] = to
    headers["Subject"] = subject
    headers["MIME-Version"] = "1.0"
    headers["Content-Type"] = "text/html; charset=UTF-8"

    // Tạo message
    var message string
    for k, v := range headers {
        message += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    message += "\r\n" + htmlBody

    // Xác thực với SMTP server
    auth := smtp.PlainAuth("", from, password, smtpHost)

    // Gửi email
    err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(message))
    if err != nil {
        log.Printf("Error sending email: %v", err)
        return err
    }
    //a
    log.Printf("Email sent to %s successfully", to)
    return nil
}