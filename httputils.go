package phx

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// FileServer creates using a router, a url path and a file path
// a file server.
func (mux *Mux) FileServer(pattern, root string) {
	fs := http.FileServer(http.Dir(root))
	http.Handle(pattern, fs)
}

// FileServerStatic creates a file server with your desired path
// a file server that serves files in ./static folder.
func (mux *Mux) FileServerStatic(path string) {
	mux.FileServer(path, "./static/")
}

// PrintLogo takes a file path and prints your fancy ascii logo.
// It will fail if your file is not found.
func PrintLogo(logoFile string) {
	logo, err := os.ReadFile(logoFile)
	if err != nil {
		log.Fatalf("Cannot read logo file: %s\n", err)
	}
	fmt.Println(string(logo))
}

func startServer(server *http.Server) {
	log.Println("Listening on address: ", server.Addr)
	log.Println("You are ready to GO!")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln("Error while listening: ", err)
	}
}

func waitAndStopServer(server *http.Server) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	log.Print("Server Stopped")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed:%+v", err)
	}

	log.Print("Server exited properly")
}

func (mux Mux) ListenAndServe(address string) {
	server := http.Server{
		Addr:    address,
		Handler: mux.router,
	}
	go startServer(&server)
	waitAndStopServer(&server)
}

func Redirect(to string) http.HandlerFunc {
	return http.RedirectHandler(to, http.StatusTemporaryRedirect).ServeHTTP
}
