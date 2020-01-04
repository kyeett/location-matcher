package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	locationmodel "github.com/kyeett/location-matcher/internal/model"
	"github.com/kyeett/location-matcher/internal/redisrepo"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Handlers struct {
	repo *redisrepo.LocationServer
}

func (h *Handlers) getDistancesHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	type getDistanceRequest struct {
		User string `json:"user"`
	}
	var req getDistanceRequest

	if user == "" {

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("failed to decode request: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		req.User = user
	}

	if req.User == "" {
		fmt.Printf("invalid request: user empty\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	distances, err := h.repo.GetDistancesFrom(req.User, 10000*1000)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(distances); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) getPositionsHandler(w http.ResponseWriter, r *http.Request) {
	pos, err := h.repo.GetAllPositions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(pos); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) postPositionHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	type postPositionRequest struct {
		User string `json:"user"`
		locationmodel.Position
	}

	var req postPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("failed to decode request: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.User == "" {
		fmt.Printf("invalid request: user empty\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err := h.repo.SetPosition(req.User, req.Position)
	if err != nil {
		fmt.Printf("failed to set position: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Printf("successfully update position of %s to (%0.2f, %0.2f)\n", req.User, req.Longitude, req.Latitude)

	w.WriteHeader(http.StatusOK)
}

func main() {
	port := os.Getenv("PORT")
	redisURL := os.Getenv("REDIS_URL")
	redisKey := os.Getenv("REDIS_KEY")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if redisURL == "" {
		log.Fatal("$REDIS_URL must be set")
	}

	if redisKey == "" {
		log.Fatal("$REDIS_KEY must be set")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	r.Mount("/static", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	repo, err := redisrepo.New(redisURL, redisKey)
	if err != nil {
		log.Fatal(err)
	}

	h := &Handlers{repo}

	r.Get("/distances", h.getDistancesHandler)
	r.Get("/positions", h.getPositionsHandler)
	r.Post("/positions", h.postPositionHandler)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
