module cortex

go 1.25

require (
    github.com/gin-gonic/gin v0.0.0
    github.com/google/gopacket v1.1.19
    github.com/redis/go-redis/v9 v9.0.0
)

require golang.org/x/sys v0.37.0 // indirect

replace github.com/gin-gonic/gin => ./third_party/gin
replace github.com/redis/go-redis/v9 => ./third_party/redis/go-redis/v9
