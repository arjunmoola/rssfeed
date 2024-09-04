package main

import (
    //"fmt"
    "log"
    "os"
    "net/http"
    "github.com/joho/godotenv"
    "encoding/json"
)

type errResponse struct {
    Error string `json:"error"`
}

type readinessMsg struct {
    Status string `json:"status"`
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
    msg := readinessMsg{ Status: "ok" }

    respondWithJSON(w, http.StatusOK, msg)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {

    respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)

    data, _ := json.Marshal(payload)

    w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)

    errMsg := errResponse{ Error: msg }

    data, _ := json.Marshal(errMsg)

    w.Write(data)

}

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal(err)
    }

    portVal := os.Getenv("PORT")


    mux := http.NewServeMux()

    mux.HandleFunc("GET /v1/healthz", readinessHandler)
    mux.HandleFunc("GET /v1/err", errorHandler)

    server := http.Server{
        Handler: mux,
        Addr: "localhost:" + portVal,
    }

    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }

}
