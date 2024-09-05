package main

import (
    //"fmt"
    "time"
    "log"
    "os"
    "net/http"
    "github.com/joho/godotenv"
    "encoding/json"
    "github.com/lib/pq"
    "database/sql"
    "github.com/arjunmoola/rssfeed/internal/database"
    "github.com/google/uuid"
    "context"
)

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func hasPrefix(text, pattern string) bool {
    return len(pattern) <= len(text) && text[:len(pattern)] == pattern
}

type errResponse struct {
    Error string `json:"error"`
}

type readinessMsg struct {
    Status string `json:"status"`
}

type createUserReqMsg struct {
    Name string `json:"name"`
}

type getUserInfoMsg struct {
    Name string `json:"name"`
}

type feedReqMsg struct {
    Name string `json:"name"`
    URL string `json:"url"`
}

type feedFollowReqMsg struct { 
    FeedId string `json:"feed_id"`
}

type deleteFeedFollowMsg struct {
    FeedFollowId string `json:"feed_follow_id"`
}

type apiConfig struct {
    DB *database.Queries
}

type CreateFeedResponseMsg struct {
    Feed database.Feed `json:"feed"`
    FeedFollow database.FeedFollow `json:"feed_follow"`
}

func (c *apiConfig) middlewareAuth(handler authedHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Println("processing request")

        apiKey := r.Header.Get("Authorization")

        if apiKey == "" {
            respondWithError(w, http.StatusOK, "Must provide an api key")
            return
        }

        if !hasPrefix(apiKey, "Bearer ") {
            respondWithError(w, http.StatusOK, "incorrect authentication")
            return
        }

        apiKey = apiKey[len("Bearer "):]

        user, err := c.DB.GetUser(context.Background(), apiKey)

        if err != nil {
            log.Println(err)
            respondWithError(w, http.StatusOK, "user does not exist")
            return
        }

        log.Println("authentication complete")

        handler(w, r, user)

    }
}

func (c *apiConfig) createFeedFollowAuth(w http.ResponseWriter, r *http.Request, user database.User) {
    log.Println("creating a feed follow")

    var reqMsg feedFollowReqMsg

    if err := json.NewDecoder(r.Body).Decode(&reqMsg); err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal server error")
        return
    }

    parsedUUID, err := uuid.Parse(reqMsg.FeedId)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal server error")
        return
    }

    feedFollowParams:= database.CreateFeedFollowParams{
        ID: uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        UserID: user.ID,
        FeedID: parsedUUID,
    }


    createdFeedFollow, err := c.DB.CreateFeedFollow(context.Background(), feedFollowParams)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal Server error")
        return
    }

    respondWithJSON(w, http.StatusOK, createdFeedFollow)

    log.Println("successfully created feed follow")
}

func (c *apiConfig) deleteFeedFollow(w http.ResponseWriter, r *http.Request, user database.User) {
    id := r.PathValue("feed_follows")

    if id == "" {
        log.Println("no feed follow id has been provided")
        respondWithError(w, http.StatusOK, "no id has been provded")
        return 
    }


    //var reqmsg deleteFeedFollowMsg

    //if err := json.NewDecoder(r.Body).Decode(&reqmsg); err != nil {
    //    log.Println(err)
    //    respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
    //    return
    //}

    parsedId, err := uuid.Parse(id)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    deletedFeedFollow, err := c.DB.DeleteFeedFollowWithId(context.Background(), parsedId)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusOK, "Unable to delete requested feed follow")
        return
    }

    respondWithJSON(w, http.StatusOK, deletedFeedFollow)

    log.Printf("Feed Follow: %v has been deleted", deletedFeedFollow.ID)
}

func (c *apiConfig) getFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
    log.Println("process get feed follows request")

    feedFollows, err := c.DB.GetFeedFollowsWithUserId(context.Background(), user.ID)
    
    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusOK, "unable to get feed follows")
        return
    }

    log.Println("select query successful")

    respondWithJSON(w, http.StatusOK, feedFollows)

    log.Println("successfully processesed requests")

}

func (c *apiConfig) createFeedAuth(w http.ResponseWriter, r *http.Request, user database.User) {
    log.Printf("creating feed for user: %v", user.ID)

    var reqMsg feedReqMsg

    if err := json.NewDecoder(r.Body).Decode(&reqMsg); err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    feedsParam := database.CreateFeedParams{
        ID: uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Name: reqMsg.Name,
        Url: reqMsg.URL,
        UserID: user.ID,
    }

    createdFeed, err := c.DB.CreateFeed(context.Background(), feedsParam)

    if err != nil {
        errName := ""
        errClass := ""
        if err , ok := err.(*pq.Error); ok {
            errName = err.Code.Name()
            errClass = err.Code.Class().Name()
        }
        log.Printf("error: pq error name:%s pq error class: %s", errName, errClass)

        if errName == "unique_violation" {
            respondWithError(w, http.StatusOK, "duplicate url")
            return
        }

        respondWithError(w, http.StatusOK, "Unable to create feed")
        return
    }

    log.Println("creating feed follow for the newly created feed")

    feedFollowParams := database.CreateFeedFollowParams {
        ID: uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        UserID: user.ID,
        FeedID: createdFeed.ID,
    }

    createdFeedFollow, err := c.DB.CreateFeedFollow(context.Background(), feedFollowParams)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusOK, "Unable to create feed follow")
        return
    }

    log.Println("created feed follow")

    responseMsg := CreateFeedResponseMsg {
        createdFeed,
        createdFeedFollow,
    }

    respondWithJSON(w, http.StatusOK, responseMsg)

    log.Println("feed and feed follow successfully created")

}

func (c *apiConfig) getFeeds(w http.ResponseWriter, r *http.Request) {
    log.Println("processing get feeds request")

    feeds, err := c.DB.GetFeeds(context.Background())
    if err != nil {
        errName := ""
        errClass := ""
        if err, ok := err.(*pq.Error); ok {
            errName = err.Code.Name()
            errClass = err.Code.Class().Name()
        }
        log.Printf("error: pq error name:%s pq error class: %s", errName, errClass)
        respondWithError(w, http.StatusInternalServerError, "internal server error")
    }

    respondWithJSON(w, http.StatusOK, feeds)

    log.Println("get feeds request successfully processed")

}

func (c *apiConfig) getUserHandlerAuth(w http.ResponseWriter, r *http.Request, user database.User) {
    respondWithJSON(w, http.StatusOK, user)
}

func (c *apiConfig) getUser(w http.ResponseWriter, r *http.Request) {
    var reqMsg getUserInfoMsg

    if err := json.NewDecoder(r.Body).Decode(&reqMsg); err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    apiKey := r.Header.Get("Authorization")

    if apiKey == "" {
        respondWithError(w, http.StatusOK, "Must provide an api key")
        return
    }

    if !hasPrefix(apiKey, "Bearer ") {
        respondWithError(w, http.StatusOK, "Incorrect authorization")
    }

    apiKey = apiKey[len("Bearer "):]

    user, err := c.DB.GetUser(context.Background(), apiKey)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    respondWithJSON(w, http.StatusOK, user)

    log.Println(user)

    //log.Println(apiKey)

}

func (c *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
    var reqMsg createUserReqMsg

    if err := json.NewDecoder(r.Body).Decode(&reqMsg); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
        return
    }

    log.Printf("processing create new user request for %v\n", reqMsg.Name)

    userParam := database.CreateUserParams{
        ID: uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Name: reqMsg.Name,
    }

    //log.Printf("created new user params %v\n", userParam)

    createdUser, err := c.DB.CreateUser(context.Background(), userParam)

    if err != nil {
        log.Println(err)
        respondWithError(w, http.StatusInternalServerError, "Unable to create new user")
        return
    }

    log.Printf("created new user params %v\n", createdUser)

    respondWithJSON(w, http.StatusCreated, createdUser)
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

    dbURL := os.Getenv("CONN")
    
    db, err := sql.Open("postgres", dbURL)

    if err != nil {
        log.Fatal(err)
    }

    dbQueries := database.New(db)

    config := apiConfig {
        DB: dbQueries,
    }

    portVal := os.Getenv("PORT")

    mux := http.NewServeMux()

    mux.HandleFunc("GET /v1/healthz", readinessHandler)
    mux.HandleFunc("GET /v1/err", errorHandler)
    mux.HandleFunc("POST /v1/users", config.createUserHandler)
    mux.HandleFunc("GET /v1/users", config.middlewareAuth(config.getUserHandlerAuth))
    mux.HandleFunc("POST /v1/feeds", config.middlewareAuth(config.createFeedAuth))
    mux.HandleFunc("POST /v1/feed_follows", config.middlewareAuth(config.createFeedFollowAuth))
    mux.HandleFunc("GET /v1/feed_follows", config.middlewareAuth(config.getFeedFollows))
    mux.HandleFunc("DELETE /v1/feed_follows/{feed_follows}", config.middlewareAuth(config.deleteFeedFollow))
    mux.HandleFunc("GET /v1/feeds", config.getFeeds)

    server := http.Server{
        Handler: mux,
        Addr: "localhost:" + portVal,
    }

    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
