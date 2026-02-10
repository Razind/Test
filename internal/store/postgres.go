package store

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateUser(email, passwordHash string) (User, error) {
	user := User{
		ID:           NewID(),
		Email:        email,
		PasswordHash: passwordHash,
	}
	_, err := s.db.Exec(
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID,
		user.Email,
		user.PasswordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) GetUserByEmail(email string) (User, error) {
	var user User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) GetUser(id string) (User, error) {
	var user User
	err := s.db.QueryRow(
		`SELECT id, email, password_hash FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) SaveLink(link Link) {
	_, _ = s.db.Exec(
		`INSERT INTO links (code, url, user_id, clicks) VALUES ($1, $2, $3, $4)`,
		link.Code,
		link.URL,
		link.UserID,
		link.Clicks,
	)
}

func (s *PostgresStore) GetLink(code string) (Link, error) {
	var link Link
	err := s.db.QueryRow(
		`SELECT code, url, user_id, clicks FROM links WHERE code = $1`,
		code,
	).Scan(&link.Code, &link.URL, &link.UserID, &link.Clicks)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		return Link{}, err
	}
	return link, nil
}

func (s *PostgresStore) ListLinksByUser(userID string) []Link {
	rows, err := s.db.Query(
		`SELECT code, url, user_id, clicks FROM links WHERE user_id = $1 ORDER BY code`,
		userID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	links := make([]Link, 0)
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.Code, &link.URL, &link.UserID, &link.Clicks); err != nil {
			continue
		}
		links = append(links, link)
	}
	return links
}

func (s *PostgresStore) DeleteLink(code, userID string) error {
	result, err := s.db.Exec(
		`DELETE FROM links WHERE code = $1 AND user_id = $2`,
		code,
		userID,
	)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		return nil
	}
	var owner string
	err = s.db.QueryRow(`SELECT user_id FROM links WHERE code = $1`, code).Scan(&owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if owner != userID {
		return ErrUnauthorized
	}
	return nil
}

func (s *PostgresStore) IncrementClick(code string) error {
	result, err := s.db.Exec(`UPDATE links SET clicks = clicks + 1 WHERE code = $1`, code)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}
