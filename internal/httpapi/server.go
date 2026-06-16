package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/keepdevops/cofiswarm-convert/internal/jobs"
)

type Server struct{ q *jobs.Queue }

func New(q *jobs.Queue) *Server { return &Server{q: q} }

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/models/convert", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			HFRepo     string `json:"hf_repo"`
			OutputName string `json:"output_name"`
			QBits      int    `json:"q_bits"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
			body.HFRepo == "" || body.OutputName == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "hf_repo and output_name required"})
			return
		}
		if body.QBits != 8 {
			body.QBits = 4
		}
		j := s.q.Create(body.HFRepo, body.OutputName, body.QBits)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": j.ID, "status": j.Status})
	})
	mux.HandleFunc("/api/models/convert/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		id := r.URL.Path[len("/api/models/convert/"):]
		j, ok := s.q.Get(id)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unknown job"})
			return
		}
		_ = json.NewEncoder(w).Encode(j)
	})
	return mux
}
