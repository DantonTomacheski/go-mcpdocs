package auth

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// UserStore provides an in-memory user storage mechanism
// In a production environment, you would replace this with a database implementation
type UserStore struct {
	users map[string]*User
	mutex sync.RWMutex
}

// NewUserStore creates a new user store
func NewUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]*User),
	}
}

// GetUser retrieves a user by username
func (s *UserStore) GetUser(username string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateUser adds a new user to the store
func (s *UserStore) CreateUser(username, email, password, role string) (*User, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if user already exists
	if _, exists := s.users[username]; exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:       generateID(), // Simple ID generation function
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
		Created:  time.Now(),
	}

	s.users[username] = user
	return user, nil
}

// AuthenticateUser checks if the provided credentials are valid
func (s *UserStore) AuthenticateUser(username, password string) (*User, error) {
	user, err := s.GetUser(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Compare the provided password with the hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GetAllUsernames returns all usernames in the store
func (s *UserStore) GetAllUsernames() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	usernames := make([]string, 0, len(s.users))
	for username := range s.users {
		usernames = append(usernames, username)
	}
	
	return usernames
}

// generateID creates a simple unique ID
// In a real application, use a more robust ID generation mechanism
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of the specified length
func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond) // Very simple way to ensure randomness
	}
	return string(result)
}
