package phx

import (
	"log"
	"net/http"
)

func logHttp(req *http.Request, res http.ResponseWriter, additional string, params ...any) {
	log.Printf("[PHX] [%s] request from %s to (%s) %s", res.Header().Get("status"), req.RemoteAddr, req.Method, req.RequestURI)
	log.Printf(additional, params...)
	log.Printf("\n")
}
