package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	gomail "gopkg.in/gomail.v2"

	"sender/cryptoRandom"
)

type Email struct {
	Email string `json:"email"`
}

func main() {
	time.Sleep(10 * time.Second)

	for {
		log.Println("sender: start daily job")
		runOnce()
		log.Println("sender: job finished, sleep 24h")

		time.Sleep(24 * time.Hour)
	}
}

func runOnce() {
	req, err := http.NewRequest("GET", os.Getenv("USERS_API_URL"), nil)
	if err != nil {
		log.Println("request build error:", err)
		return
	}
	req.Header.Set("X-Internal-Token", os.Getenv("INTERNAL_SERVICE_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("request error:", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Println("Response body:\n", string(bodyBytes))
	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var users []Email
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		log.Println("JSON decode error:", err)
		return
	}
	log.Println("Parsed users:", users)

	password, err := cryptoRandom.AsciiString(12)
	if err != nil {
		log.Println("Password generation error:", err)
		return
	}
	log.Println("Entry password generated:", password)

	setURL := strings.TrimSpace(os.Getenv("PASSWORD_SET_URL"))
	if setURL != "" {
		if err := postJSON(setURL, map[string]string{"password": password}, os.Getenv("INTERNAL_SERVICE_TOKEN")); err != nil {
			log.Println("Set password API error:", err)
		} else {
			log.Println("Password stored")
		}
	}

	d := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		587,
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
	)

	for _, u := range users {
		if u.Email == "" {
			continue
		}
		m := gomail.NewMessage()
		m.SetHeader("From", os.Getenv("SMTP_USER"))
		m.SetHeader("To", u.Email)
		m.SetHeader("Subject", "👁️ The Illuminators Message")
		m.SetBody("text/plain",
			fmt.Sprintf("Brother of the Light,\n\n🗝️: %s\n\nKeep it safe.", password))

		if err := d.DialAndSend(m); err != nil {
			log.Println("Send error:", err)
		} else {
			log.Println("Email sent to:", u.Email)
		}
	}
}

func postJSON(url string, body any, token string) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Internal-Token", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API %s returned %s: %s", url, resp.Status, string(rb))
	}
	return nil
}