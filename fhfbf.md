docker exec -it local_postgres psql -U myuser -d myapp -c "SELECT * FROM users;"

curl.exe -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{\"name\":\"Анна\",\"email\":\"anna@example.com\"}'


Invoke-WebRequest -Uri http://localhost:8080/users -Method POST -ContentType "application/json" -Body '{"name":"Alexey","email":"alex@gmail.com"}'


Invoke-WebRequest -Uri http://localhost:8080/users/1
