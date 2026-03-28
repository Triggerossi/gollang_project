package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Triggerossi/gollang_project/repo"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

var db *sql.DB // ← ГЛОБАЛЬНАЯ переменная db (исправление главной ошибки)

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

const secret = "mysecretkey"
const accessTokenDuration = 15 * time.Minute
const refreshTokenDuration = 7 * 24 * time.Hour

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeerror(w, http.StatusUnauthorized, "missing_token", "токен отсутствует")
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeerror(w, http.StatusUnauthorized, "invalid_token", "неверный формат токена")
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			writeerror(w, http.StatusUnauthorized, "invalid_token", "токен невалидный или просрочен")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeerror(w, http.StatusUnauthorized, "invalid_token", "неверный токен")
			return
		}
		sub, ok := claims["sub"].(string)
		if !ok {
			writeerror(w, http.StatusUnauthorized, "invalid_token", "неверный токен")
			return
		}
		ctx := context.WithValue(r.Context(), "userID", sub)
		r = r.WithContext(ctx)
		next(w, r)
	}
}

func dataRaceDemo() {
	var counter int
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++
		}()
	}

	wg.Wait()
	fmt.Println("dataRaceDemo counter:", counter)
}

func dataRaceDemoFixed() {
	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}

	wg.Wait()
	fmt.Println("dataRaceDemoFixed counter:", counter)
}

type WorkTask struct {
	Input string
}

func worker(id int, tasks <-chan WorkTask, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		hash := sha256.Sum256([]byte(task.Input))
		results <- fmt.Sprintf("worker %d: %s -> %x", id, task.Input, hash[:6])
		time.Sleep(10 * time.Millisecond)
	}
}

func runWorkerPool(taskCount, numWorkers int) []string {
	tasks := make(chan WorkTask)
	results := make(chan string, taskCount)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, tasks, results, &wg)
	}

	go func() {
		for i := 0; i < taskCount; i++ {
			tasks <- WorkTask{Input: fmt.Sprintf("task-%d", i)}
		}
		close(tasks)
	}()

	wg.Wait()
	close(results)

	out := make([]string, 0, taskCount)
	for r := range results {
		out = append(out, r)
	}
	return out
}

// ====================== АСИНХРОННАЯ ИНДЕКСАЦИЯ ======================

const maxQueueSize = 50

type IndexTask struct {
	NoteID int64
	Text   string
}

var indexQueue = make(chan IndexTask, maxQueueSize)

// Запуск индексатора
func startIndexer() {
	go func() {
		fmt.Println("✅ Асинхронный индексатор запущен (очередь:", maxQueueSize, "задач)")
		for task := range indexQueue {
			processIndexing(task)
		}
	}()
}

func processIndexing(task IndexTask) {
	wordCount := countWords(task.Text)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx,
		"UPDATE notes SET word_count = $1 WHERE id = $2",
		wordCount, task.NoteID)

	if err != nil {
		fmt.Printf("❌ Ошибка индексации note %d: %v\n", task.NoteID, err)
	} else {
		fmt.Printf("✅ Индексация завершена: note %d → %d слов\n", task.NoteID, wordCount)
	}
}

func countWords(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(strings.Fields(text))
}

// ====================== ХЕНДЛЕР СОЗДАНИЯ ЗАМЕТКИ ======================

func createNoteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeerror(w, http.StatusBadRequest, "bad_json", "Некорректный JSON")
		return
	}

	if req.Title == "" || req.Text == "" {
		writeerror(w, http.StatusBadRequest, "validation_error", "Поля title и text обязательны")
		return
	}

	// Создаём заметку
	var noteID int64
	err := db.QueryRowContext(r.Context(),
		`INSERT INTO notes (title, text) VALUES ($1, $2) RETURNING id`,
		req.Title, req.Text).Scan(&noteID)

	if err != nil {
		fmt.Println("Ошибка создания заметки:", err)
		writeerror(w, http.StatusInternalServerError, "db_error", "Не удалось создать заметку")
		return
	}

	// Отправляем задачу на индексацию
	select {
	case indexQueue <- IndexTask{NoteID: noteID, Text: req.Text}:
		// успешно
	default:
		writeerror(w, http.StatusServiceUnavailable, "queue_full",
			"Сервер перегружен. Попробуйте создать заметку позже.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      noteID,
		"title":   req.Title,
		"message": "Заметка создана. Подсчёт слов выполняется в фоне.",
	})
}

// ====================== MAIN ======================
func main() {
	connStr := "host=localhost port=5432 user=myuser password=mypassword dbname=myapp sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr) // ← исправлено: присваиваем глобальной db
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Подключение к PostgreSQL успешно")

	userRepo := repo.NewUserRepo(db)

	dataRaceDemo()
	dataRaceDemoFixed()
	poolRes := runWorkerPool(100, 10)
	fmt.Printf("Worker pool done, processed=%d\n", len(poolRes))

	startIndexer()

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

	mux.HandleFunc("/concurrent", func(w http.ResponseWriter, r *http.Request) {
		type result struct {
			DB   string `json:"db"`
			API  string `json:"api"`
			Time string `json:"time"`
		}

		dbCh := make(chan string, 1)
		apiCh := make(chan string, 1)

		go func() {
			time.Sleep(100 * time.Millisecond)
			dbCh <- "db result"
		}()

		go func() {
			time.Sleep(150 * time.Millisecond)
			apiCh <- "external result"
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result{
			DB:   <-dbCh,
			API:  <-apiCh,
			Time: time.Now().Format(time.RFC3339Nano),
		})
	})

	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, 400, "bad_json", "плохой json")
			return
		}
		if req.Email != "user@example.com" || req.Password != "password" {
			writeerror(w, http.StatusUnauthorized, "invalid_credentials", "неверные учетные данные")
			return
		}

		accessClaims := jwt.MapClaims{
			"sub": "1",
			"exp": time.Now().Add(accessTokenDuration).Unix(),
		}
		accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
		accessTokenString, err := accessToken.SignedString([]byte(secret))
		if err != nil {
			writeerror(w, http.StatusInternalServerError, "token_error", "ошибка генерации токена")
			return
		}

		refreshClaims := jwt.MapClaims{
			"sub": "1",
			"exp": time.Now().Add(refreshTokenDuration).Unix(),
		}
		refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
		refreshTokenString, err := refreshToken.SignedString([]byte(secret))
		if err != nil {
			writeerror(w, http.StatusInternalServerError, "token_error", "ошибка генерации токена")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  accessTokenString,
			"refresh_token": refreshTokenString,
		})
	})

	mux.HandleFunc("POST /auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeerror(w, 400, "bad_json", "плохой json")
			return
		}
		token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			writeerror(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh токен невалидный или просрочен")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeerror(w, http.StatusUnauthorized, "invalid_refresh_token", "неверный refresh токен")
			return
		}
		sub, ok := claims["sub"].(string)
		if !ok {
			writeerror(w, http.StatusUnauthorized, "invalid_refresh_token", "неверный refresh токен")
			return
		}

		accessClaims := jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(accessTokenDuration).Unix(),
		}
		accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
		accessTokenString, err := accessToken.SignedString([]byte(secret))
		if err != nil {
			writeerror(w, http.StatusInternalServerError, "token_error", "ошибка генерации токена")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": accessTokenString})
	})

	mux.HandleFunc("GET /users/{id}", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
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
	}))

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

	mux.HandleFunc("PUT /users/{id}", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		idstr := r.PathValue("id")
		id, err := strconv.ParseInt(idstr, 10, 64)
		if err != nil {
			writeerror(w, http.StatusBadRequest, "invalid_id", "id должен быть числом")
			return
		}

		userIDStr := r.Context().Value("userID").(string)
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			writeerror(w, http.StatusUnauthorized, "invalid_user", "неверный пользователь")
			return
		}
		if userID != id {
			writeerror(w, http.StatusForbidden, "forbidden", "нет прав на редактирование")
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
	}))

	// Новый хендлер для заметок
	mux.HandleFunc("POST /notes", createNoteHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("Starting server at %s\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server failed: %v\n", err)
	}
}
