package files

import "net/http"

var Handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
})
