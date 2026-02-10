package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type UserRow struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CPF          string
	BirthDate    string
	PhotoURL     sql.NullString
	Role         string
	CreatedAt    time.Time
}

func parseCreatedAt(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func UserByEmail(db *sql.DB, email string) (*UserRow, error) {
	var u UserRow
	var createdAt sql.NullString
	err := db.QueryRow(`SELECT id, name, email, password_hash, cpf, birth_date, photo_url, role, created_at FROM users WHERE email = ?`, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CPF, &u.BirthDate, &u.PhotoURL, &u.Role, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		u.CreatedAt = parseCreatedAt(createdAt.String)
	}
	return &u, nil
}

func UserByID(db *sql.DB, id string) (*UserRow, error) {
	var u UserRow
	var createdAt sql.NullString
	err := db.QueryRow(`SELECT id, name, email, password_hash, cpf, birth_date, photo_url, role, created_at FROM users WHERE id = ?`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CPF, &u.BirthDate, &u.PhotoURL, &u.Role, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if createdAt.Valid {
		u.CreatedAt = parseCreatedAt(createdAt.String)
	}
	return &u, nil
}

func CreateUser(db *sql.DB, name, email, passwordHash, cpf, birthDate string) (string, error) {
	id := uuid.New().String()
	_, err := db.Exec(`INSERT INTO users (id, name, email, password_hash, cpf, birth_date, role) VALUES (?, ?, ?, ?, ?, ?, 'USER')`,
		id, name, email, passwordHash, cpf, birthDate,
	)
	return id, err
}

func UpdateUserPhotoURL(db *sql.DB, userID, photoURL string) error {
	_, err := db.Exec(`UPDATE users SET photo_url = ? WHERE id = ?`, photoURL, userID)
	return err
}
