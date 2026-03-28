docker exec -it local_postgres psql -U myuser -d myapp -c "SELECT * FROM users;"

curl.exe -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{\"name\":\"Анна\",\"email\":\"anna@example.com\"}'


Invoke-WebRequest -Uri http://localhost:8080/users -Method POST -ContentType "application/json" -Body '{"name":"Alexey","email":"alex@gmail.com"}'


Invoke-WebRequest -Uri http://localhost:8080/users/1



eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzQxNzA0MTksInN1YiI6IjEifQ.etu0Za8gZNoGERNNTQhJdi4TnagOumYv9-RcN6x39JI

Invoke-WebRequest -Uri "http://localhost:8080/users/1" -Method GET -Headers @{ "Authorization" = "Bearer YOUR_TOKEN" }


Invoke-WebRequest -Uri "http://localhost:8080/users/1" -Method GET -Headers @{ "Authorization" = "Bearer invalid_token" }


Invoke-WebRequest -Uri "http://localhost:8080/users/1" -Method GET




(Invoke-WebRequest -Uri "http://localhost:8080/auth/login" -Method POST -Headers @{ "Content-Type" = "application/json" } -Body '{"email":"user@example.com","password":"password"}').Content



eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzQwODU3MzQsInN1YiI6IjEifQ.7IKUHGJKlxKuTLWycwhh9cPh-OHOzObbvbRcv5IbB08

eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzQ2ODk2MzQsInN1YiI6IjEifQ.uUh9081MKeFX8w5BBsKoeedaTv1_HB6apIkjBDHQ1fw


Invoke-WebRequest -Uri "http://localhost:8080/users/1" -Method GET -Headers @{ "Authorization" = "Bearer YOUR_ACCESS_TOKEN" }

Invoke-WebRequest -Uri "http://localhost:8080/users/1" -Method PUT -Headers @{ "Authorization" = "Bearer YOUR_ACCESS_TOKEN"; "Content-Type" = "application/json" } -Body '{"name":"New Name","email":"new@example.com"}'


Invoke-WebRequest -Uri "http://localhost:8080/auth/refresh" -Method POST -Headers @{ "Content-Type" = "application/json" } -Body '{"refresh_token":"YOUR_REFRESH_TOKEN"}'


eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzQwODcwNTgsInN1YiI6IjEifQ.FppVEy-uUdfX4GGdLiKae8gdTDh3gMM6EOEwG6_KtCk

eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NzQ2OTA5NTgsInN1YiI6IjEifQ.YkbsSDzMzGAPFejvESGEtY7F3Icf5jgb48r9nv41Hpg