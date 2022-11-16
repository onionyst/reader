package main

import (
	"bufio"
	"fmt"
	"os"

	"reader/internal/app/reader/db"
	"reader/internal/app/reader/models"
	"reader/internal/pkg/utils"
)

func getEmailAndPassword() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	email = email[:len(email)-1]

	fmt.Print("Password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	password = password[:len(password)-1]

	return email, password
}

func addAccount(email, password string) error {
	password, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = models.AddUser(email, password)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	fmt.Println("OnionReader Account Manager")

	pg := db.SetupDatabase()
	defer db.CloseDatabase(pg)

	fmt.Println("Add account: please input email and password.")
	email, password := getEmailAndPassword()

	err := addAccount(email, password)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully added account for %s\n", email)
}
