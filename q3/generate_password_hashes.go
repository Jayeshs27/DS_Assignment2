package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
)

// User struct for JSON storage
type User struct {
	Username     string `json:"username"`
	Password string `json:"password"`
	Role string `json:"role"`
	AccountNo  string `json:"account_no"`
	Balance    float64 `json:"balance"`
}

// JSON file path

const userFile = "users.json"

// Load users from JSON file
func loadUsers() ([]User, error) {
	file, err := os.ReadFile(userFile)
	if err != nil {
		// If file doesn't exist, return an empty list
		if os.IsNotExist(err) {
			return []User{}, nil
		}
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(file, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// Save users to JSON file
func saveUsers(users []User) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(userFile, data, 0644)
}

// Hash password and store in JSON
func hashAndStorePassword(username, password string) {
	users, _ := loadUsers()
	// Check if user already exists
	for _, user := range users {
		if user.Username == username {
			fmt.Println("User already exists!")
			return
		}
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		return
	}

	// Add new user to the list
	newUser := User{Username: username, 
		Password: string(hashedPassword),
		Role: "customer",
		AccountNo: "1234",
		Balance: 400.0,
	}
	users = append(users, newUser)

	// Save updated users to JSON
	if err := saveUsers(users); err != nil {
		fmt.Println("Error saving users:", err)
		return
	}

	fmt.Printf("Stored user: %s\n", username)
}

// Verify user login
func verifyPassword(username, enteredPassword string) bool {
	users, _ := loadUsers() // Load users from JSON

	// Find user
	for _, user := range users {
		if user.Username == username {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(enteredPassword))
			if err != nil {
				fmt.Println("Authentication failed!")
				return false
			}
			fmt.Println("Authentication successful!")
			return true
		}
	}

	fmt.Println("User not found!")
	return false
}

func main() {

	hashAndStorePassword("alice", "password123")
	hashAndStorePassword("bob", "MySafeP@ss2")
	hashAndStorePassword("charlie", "TopSecret#3")

	fmt.Println("\n🔍 Verifying Passwords:")
	verifyPassword("alice", "password123")  
	verifyPassword("alice", "WrongPass!")    
	verifyPassword("bob", "MySafeP@ss2")     
}
