package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/Triggerossi/gollang_project/repo"
	_ "github.com/lib/pq"
)

func validateRequired(value string, fieldname string) error {
	trim := strings.TrimSpace(value)
	if trim == "" {
		return fmt.Errorf("поле %s пустое", fieldname)
	}
	return nil
}

func validatelen(value string, fieldname string) error {
	trim := strings.TrimSpace(value)
	len1 := len(trim)
	if len1 < 2 || len1 > 27 {
		return fmt.Errorf("либо слишком длиное либо слишком короткео поле %s", fieldname)
	}
	return nil
}

func validateemail(email string, fieldname string) error {
	if email == "" {
		return fmt.Errorf("поле должно быть заполнено %s", fieldname)
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("поле %s ошибочно", fieldname)
	}
	return nil
}

type ApiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Requestid string `json:"requestid"`
}

type ErrorResponse struct {
	Error ApiError `json:"error"`
}

func writeerror(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ErrorResponse{
		Error: ApiError{
			Code:    code,
			Message: msg,
		},
	}

	b := make([]byte, 8)
	if _, err := rand.Read(b); err == nil {
		resp.Error.Requestid = hex.EncodeToString(b)
	}

	_ = json.NewEncoder(w).Encode(resp)
}
func main() {
	connStr := "host=localhost port=5432 user=myuser password=mypassword dbname=myapp sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	userRepo := repo.NewUserRepo(db)

	mux := http.NewServeMux()

	type Createuserrequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type User struct {
		Id    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.ParseInt(idstr, 10, 64)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "invalid_id", "id должен быть числом")
			return
		}

		user, err := userRepo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeerror(w, http.StatusNotFound, "not_found", "пользователь не найден")
				return
			}
			fmt.Println("GET error:", err)
			writeerror(w, http.StatusInternalServerError, "db_error", "ошибка базы")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, 400, "bad_json", "плохой json")
			return
		}

		id, err := userRepo.Create(r.Context(), req.Name, req.Email)
		if err != nil {
			if errors.Is(err, repo.ErrEmailExists) {
				writeerror(w, http.StatusConflict, "email_conflict", "email уже занят")
				return
			}
			fmt.Println("CREATE error:", err)
			writeerror(w, http.StatusInternalServerError, "db_error", "ошибка базы")
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    id,
			"name":  req.Name,
			"email": req.Email,
		})
	})

	mux.HandleFunc("PUT /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.ParseInt(idstr, 10, 64)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "invalid_id", "id должен быть числом")
			return
		}

		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, http.StatusBadRequest, "invalid_json", "некорректный JSON")
			return
		}

		var valErr error
		if err := validateRequired(req.Name, "name"); err != nil {
			valErr = err
		} else if err := validateRequired(req.Email, "email"); err != nil {
			valErr = err
		} else if err := validateemail(req.Email, "email"); err != nil {
			valErr = err
		} else if err := validatelen(req.Name, "name"); err != nil {
			valErr = err
		} else if err := validatelen(req.Email, "email"); err != nil {
			valErr = err
		}

		if valErr != nil {
			writeerror(w, http.StatusBadRequest, "validation_error", valErr.Error())
			return
		}

		err = userRepo.Update(r.Context(), id, req.Name, req.Email)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeerror(w, http.StatusNotFound, "not_found", "пользователь не найден")
				return
			}
			if errors.Is(err, repo.ErrEmailExists) {
				writeerror(w, http.StatusConflict, "email_conflict", "email уже занят")
				return
			}
			fmt.Println("UPDATE error:", err)
			writeerror(w, http.StatusInternalServerError, "db_error", "ошибка базы")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    id,
			"name":  req.Name,
			"email": req.Email,
		})
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("Starting server at %s\n", srv.Addr) // ← исправил printf
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server failed: %v\n", err)
	}
}
