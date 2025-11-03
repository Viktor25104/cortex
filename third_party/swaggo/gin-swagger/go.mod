module github.com/swaggo/gin-swagger

go 1.25

require (
        github.com/gin-gonic/gin v0.0.0
        github.com/swaggo/swag v0.0.0
)

replace github.com/gin-gonic/gin => ../../gin
replace github.com/swaggo/swag => ../swag
