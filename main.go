package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

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
	mux := http.NewServeMux()

	type Createuserrequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type User struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var users = map[int]User{}
	//var nextid = 1

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.Atoi(idstr)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "empty", "add somthing")
		}
		if id < 0 {
			writeerror(w, http.StatusBadRequest, "invalid id", "id need to positive")
		}

		user, found := users[id]
		if !found {
			writeerror(w, http.StatusNotFound, "user is not found", "this user didn't created")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var req Createuserrequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, http.StatusBadRequest, "invalid error", "change somrthing")
			return
		}
		for _, user := range users {
			if user.Email == req.Email {
				writeerror(w, http.StatusConflict, "email_conflict", "use another email")
				return
			}
		}

		var errs []error
		if err := validateRequired(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validateRequired(req.Name, "name"); err != nil {
			errs = append(errs, err)
		}
		if err := validateemail(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validatelen(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validatelen(req.Name, "name"); err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			writeerror(w, http.StatusBadRequest, "validation failed", "change smf")
			return
		}
		var id int
		db.QueryRow("insert into users (name, email) values ($1, $2) returning id", req.Name, req.Email).Scan(&id)
		/*
			newUser := User{
				Id:    nextid,
				Name:  req.Name,
				Email: req.Email,
			} */
		/* users[nextid] = newUser
		nextid++ */

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Id":    id,
			"Name":  req.Name,
			"Email": req.Email,
		})
	})
	mux.HandleFunc("PUT /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.Atoi(idstr)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "empty", "add somthing")
			return
		}
		if id < 0 {
			writeerror(w, http.StatusBadRequest, "invalid id", "id need to positive")
			return
		}

		user, found := users[id]
		if !found {
			writeerror(w, http.StatusNotFound, "user is not found", "this user didn't created")
			return
		}

		var req Createuserrequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, http.StatusBadRequest, "so bad json", "send good json")
			return
		}
		var errs []error
		if err := validateRequired(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validateRequired(req.Name, "name"); err != nil {
			errs = append(errs, err)
		}
		if err := validateemail(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validatelen(req.Email, "email"); err != nil {
			errs = append(errs, err)
		}
		if err := validatelen(req.Name, "name"); err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			writeerror(w, http.StatusBadRequest, "validation failed", "change smf")
			return
		}

		if user.Email != req.Email {
			for _, u := range users {
				if req.Email == u.Email {
					writeerror(w, http.StatusConflict, "this email used", "change email")
					return
				}
			}
		}

		user.Email = req.Email
		user.Name = req.Name
		users[id] = user
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})
	mux.HandleFunc("PATCH /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.Atoi(idstr)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "empty", "add somthing")
		}
		if id < 0 {
			writeerror(w, http.StatusBadRequest, "invalid id", "id need to positive")
		}

		user, found := users[id]
		if !found {
			writeerror(w, http.StatusNotFound, "user is not found", "this user didn't created")
		}

		var req Createuserrequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, http.StatusBadRequest, "so bad json", "send good json")
			fmt.Println(err)
			return
		}

		if req.Email != "" {
			if err := validateemail(req.Email, "email"); err != nil {
				writeerror(w, http.StatusBadRequest, "bad email", "send good emaul")
			}
			if err := validatelen(req.Email, "email"); err != nil {
				writeerror(w, http.StatusBadRequest, "long or short", "change len")
			}
			if user.Email != req.Email {
				for _, u := range users {
					if req.Email == u.Email {
						writeerror(w, http.StatusConflict, "this email used", "change email")
						return
					}
				}
			}
			user.Email = req.Email
		}
		if req.Name != "" {
			if err := validatelen(req.Name, "name"); err != nil {
				writeerror(w, http.StatusBadRequest, "long or short", "change len")
				return
			}
			user.Name = req.Name
		}

		users[id] = user

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Println("Starting server at: %s", srv.Addr)
	err = srv.ListenAndServe()
	if err != nil {
		fmt.Println("Failed to start server: %s", srv.Addr)
	}
}
