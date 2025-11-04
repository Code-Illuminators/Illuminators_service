package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	gomail "gopkg.in/gomail.v2"
)

type StartVoting struct {
	ID                int     `json:"id"`
	NominatedUserName string  `json:"nominated_user_username"`
	VoteType          string  `json:"vote_type"`
	PromotionRole     *string `json:"promotion_role"`
}

type DjangoUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func main() {
	http.HandleFunc("/voting/start", handleStartVoting)

	log.Println("sender/voting service on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleStartVoting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload StartVoting
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("start: bad json:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	role := "none"
	if payload.PromotionRole != nil {
		role = *payload.PromotionRole
	}

	log.Printf("start: id=%d type=%s promotion_role=%s nominated=%s",
		payload.ID, payload.VoteType, role, payload.NominatedUserName)

	users, err := getUsersFromDjango()
	if err != nil {
		log.Println("start: getUsersFromDjango error:", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	var filtered []DjangoUser
	if strings.EqualFold(payload.VoteType, "promotion") &&
		payload.PromotionRole != nil &&
		*payload.PromotionRole != "" {
		filtered = filterUsersByRoles(users, []string{*payload.PromotionRole})
		log.Printf("start: %d users match promotion_role=%s", len(filtered), *payload.PromotionRole)
	} else {
		filtered = users
		log.Printf("start: promotion_role missing → ALL users will vote: %d", len(filtered))
	}

	filtered = excludeNominated(filtered, payload.NominatedUserName)
	log.Printf("start: %d users will actually receive email (without nominee)", len(filtered))

	if len(filtered) > 0 {
		if err := sendEmail(filtered, payload.NominatedUserName); err != nil {
			log.Println("start: email send error:", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok"}`)
}

func getUsersFromDjango() ([]DjangoUser, error) {
	url := os.Getenv("USERS_API_URL")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	internalToken := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if internalToken != "" {
		req.Header.Set("X-Internal-Token", internalToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Println("Django users raw:\n", string(bodyBytes))

	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var users []DjangoUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}

	return users, nil
}

func filterUsersByRoles(users []DjangoUser, roles []string) []DjangoUser {
	if len(roles) == 0 {
		return nil
	}

	var out []DjangoUser
	for _, u := range users {
		if roleInList(u.Role, roles) {
			out = append(out, u)
		}
	}
	return out
}

func roleInList(role string, list []string) bool {
	for _, r := range list {
		if strings.EqualFold(role, r) {
			return true
		}
	}
	return false
}

func excludeNominated(users []DjangoUser, nominatedUsername string) []DjangoUser {
	if nominatedUsername == "" {
		return users
	}
	var out []DjangoUser
	for _, u := range users {
		if strings.EqualFold(u.Username, nominatedUsername) {
			continue
		}
		out = append(out, u)
	}
	return out
}

func sendEmail(users []DjangoUser, nominatedName string) error {
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
		m.SetHeader("Subject", "Start voting")
		m.SetBody("text/plain",
			fmt.Sprintf("Hi, %s!\nVoting about %s has started. Please vote.", u.Username, nominatedName),
		)

		if err := d.DialAndSend(m); err != nil {
			log.Println("Send error:", err)
		} else {
			log.Println("Email sent to:", u.Email)
		}
	}
	return nil
}