# только имя
Invoke-WebRequest -Uri http://localhost:8080/users/11 `
  -Method Put `
  -ContentType "application/json" `
  -Body '{"name":"СуперНовоеИмя"}' `
  -UseBasicParsing


# только email
Invoke-WebRequest -Uri http://localhost:8080/users/11 `
  -Method Put `
  -ContentType "application/json" `
  -Body '{"email":"changed.email@new.ru"}' `
  -UseBasicParsing


# оба поля
Invoke-WebRequest -Uri http://localhost:8080/users/11 `
  -Method Put `
  -ContentType "application/json" `
  -Body '{"name":"ПолноеИмя","email":"final.email@domain.ru"}' `
  -UseBasicParsing
  
curl.exe -X PATCH http://localhost:8080/users/8 -d '{"name":"test"}'