package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"reader/internal/app/reader/db"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

const (
	serviceTimeout = 15 // seconds
)

func readEmailAndPassword() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return "", "", errors.New("email is required")
	}

	fmt.Print("Password: ")
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", "", err
	}
	password := strings.TrimSpace(string(pwBytes))
	if password == "" {
		return "", "", errors.New("password is required")
	}

	return email, password, nil
}

func addAccount(email, password string) error {
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = models.AddUser(email, hashed)
	return err
}

func main() {
	fmt.Println("OnionReader Account Manager")

	services := []string{db.ServiceString()}
	utils.Wait(services, serviceTimeout)

	pg := db.SetupDatabase()
	defer db.CloseDatabase(pg)

	fmt.Println("Add account: please input email and password.")
	email, password, err := readEmailAndPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Input error:", err)
		os.Exit(2)
	}

	if err := addAccount(email, password); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to add account:", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully added account for %s\n", email)
}
