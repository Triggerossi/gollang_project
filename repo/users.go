package repo

import (
	"context"
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")
var ErrDatabase = errors.New("database error")

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, name, email string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (name, email)
		 VALUES ($1, $2)
		 RETURNING id`,
		name, email,
	).Scan(&id)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return 0, ErrEmailExists
		}
		return 0, errors.Join(ErrDatabase, err)
	}

	return id, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, email
		 FROM users
		 WHERE id = $1`,
		id,
	).Scan(
		&user.ID, &user.Name, &user.Email,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, errors.Join(ErrDatabase, err)
	}

	return user, nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email
		 FROM users
		 ORDER BY name DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Email,
		); err != nil {
			return nil, errors.Join(ErrDatabase, err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}

	return users, nil
}
