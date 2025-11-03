module cortex

go 1.25

require (
        github.com/gin-gonic/gin v0.0.0
        github.com/google/gopacket v1.1.19
        github.com/joho/godotenv v0.0.0
        github.com/redis/go-redis/v9 v9.0.0
        github.com/swaggo/files v0.0.0
        github.com/swaggo/gin-swagger v0.0.0
        github.com/swaggo/swag v0.0.0
)

require golang.org/x/sys v0.37.0 // indirect

replace github.com/gin-gonic/gin => ./third_party/gin
replace github.com/redis/go-redis/v9 => ./third_party/redis/go-redis/v9
replace github.com/swaggo/files => ./third_party/swaggo/files
replace github.com/swaggo/gin-swagger => ./third_party/swaggo/gin-swagger
replace github.com/swaggo/swag => ./third_party/swaggo/swag
replace github.com/joho/godotenv => ./third_party/godotenv
