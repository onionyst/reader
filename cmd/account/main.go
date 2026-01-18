package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"

	readerModels "reader/internal/app/reader/models"
	"reader/internal/pkg/db"
	"reader/internal/pkg/utils"
)

func main() {
	fmt.Println("OnionReader Account Manager")

	conns, err := db.Setup()
	if err != nil {
		log.Fatal(err)
	}
	defer conns.Close()

	if err := readerModels.Register(conns.Main); err != nil {
		log.Fatal(err)
	}

	repo := readerModels.NewRepo(conns.Main)

	fmt.Println("Add account: please input email and password.")
	email, password, err := readEmailAndPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Input error:", err)
		os.Exit(2)
	}

	if err := addAccount(context.Background(), repo, email, password); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to add account:", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully added account for %s\n", email)
}

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

func addAccount(ctx context.Context, repo *readerModels.Repo, email, password string) error {
	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = repo.AddUser(ctx, email, hashed)
	return err
}
