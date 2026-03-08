package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

var (
	ErrNotFound    = errors.New("user not found")
	ErrEmailExists = errors.New("email already exists")
	ErrDatabase    = errors.New("database error")
)

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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO users (name, email, created_at)
		 VALUES ($1, $2, NOW())
		 RETURNING id`,
		name, email,
	).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return 0, ErrEmailExists
		}
		return 0, errors.Join(ErrDatabase, err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO audit_logs (action, entity_id) VALUES ('create', $1)`,
		id,
	)
	if err != nil {
		return 0, errors.Join(ErrDatabase, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *UserRepo) Update(ctx context.Context, id int64, name, email string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err != nil {
		return errors.Join(ErrDatabase, err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users 
		 SET name = $1, email = $2
		 WHERE id = $3`,
		name, email, id,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrEmailExists
		}
		return errors.Join(ErrDatabase, err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO audit_logs (action, entity_id) VALUES ('update', $1)`,
		id,
	)
	if err != nil {
		return errors.Join(ErrDatabase, err)
	}

	return tx.Commit()
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
