 Email Sender Service (Go)

Description

This Go service automatically generates and sends secret entry passwords to users every 24 hours.
It connects to the Django backend, generates a random password, saves it through the protected API, and delivers it by email via SMTP.
All configuration values are loaded from the .env file.

 How It Works
	1.	Reads configuration from .env.
	2.	Requests the list of users from Django (USERS_API_URL).
	3.	Generates a new random password using the internal cryptoRandom package.
	4.	Stores the password in the backend (PASSWORD_SET_URL).
	5.	Sends the password to each user via email.
	6.	Repeats the process automatically every 24 hours.

Run with Docker build
```
docker compose build -t go-sender .
```
