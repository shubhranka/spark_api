package data

import (
	"database/sql"
	"errors"
	"time"
)

// User model represents a user in our database. It no longer has a password.
type User struct {
	ID          string    `json:"id"`
	FirebaseUID string    `json:"firebase_uid"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserModel wraps the database connection.
type UserModel struct {
	DB *sql.DB
}

// Insert adds a new user record to the users table.
func (m UserModel) Insert(user *User) error {
	query := `
        INSERT INTO users (firebase_uid, email, display_name)
        VALUES ($1, $2, $3)
        RETURNING id, created_at, updated_at`

	args := []interface{}{user.FirebaseUID, user.Email, user.DisplayName}

	return m.DB.QueryRow(query, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetByFirebaseUID retrieves a user by their unique Firebase ID.
func (m UserModel) GetByFirebaseUID(firebaseUID string) (*User, error) {
	query := `
        SELECT id, firebase_uid, email, display_name, created_at, updated_at
        FROM users
        WHERE firebase_uid = $1`

	var user User
	err := m.DB.QueryRow(query, firebaseUID).Scan(&user.ID, &user.FirebaseUID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows // Return the specific error
		}
		return nil, err
	}
	return &user, nil
}

// GetByID retrieves a user by their internal UUID.
func (m UserModel) GetByID(id string) (*User, error) {
	query := `
        SELECT id, firebase_uid, email, display_name, created_at, updated_at
        FROM users
        WHERE id = $1`

	var user User
	err := m.DB.QueryRow(query, id).Scan(&user.ID, &user.FirebaseUID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}
