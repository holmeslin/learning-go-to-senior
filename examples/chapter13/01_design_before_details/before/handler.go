package signup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// NewHandler 回傳活動報名的 HTTP handler：
//
//	POST   /events/{id}/registrations        本文 {"who":"u1"}：報名，201；額滿且候補也滿、或重複 409
//	DELETE /events/{id}/registrations/{who}  取消報名並遞補候補名單的第一位，204；沒有紀錄 404
//	GET    /events/{id}                      活動目前的名額、報名者與候補名單
//
// 活動不存在一律回 404。
func NewHandler(store *Store) http.Handler {
	h := &handler{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events/{id}/registrations", h.register)
	mux.HandleFunc("DELETE /events/{id}/registrations/{who}", h.cancel)
	mux.HandleFunc("GET /events/{id}", h.show)
	return mux
}

type handler struct {
	store *Store
}

// registerRequest 是報名請求的 JSON 本文。
type registerRequest struct {
	Who string `json:"who"`
}

// eventView 是 GET 的回應。
type eventView struct {
	Seats     int      `json:"capacity"`
	Confirmed []string `json:"attendees"`
	Waitlist  []string `json:"waitlist"`
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	ev, ok := h.store.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	switch err := ev.Register(req.Who); {
	case err == nil:
		w.WriteHeader(http.StatusCreated)
	case errors.Is(err, ErrFull), errors.Is(err, ErrAlreadyRegistered):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, ErrEmptyName):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// ANCHOR: cancel

func (h *handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, who := r.PathValue("id"), r.PathValue("who")
	ev, ok := h.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch err := ev.Cancel(who); {
	case err == nil:
		// 取消之後，把空出來的名額遞補給候補名單的第一位。
		ev.PromoteNext()
		// ANCHOR: debug
		fmt.Printf("DEBUG cancel: event=%s who=%s confirmed=%v waitlist=%v\n", id, who, ev.Confirmed(), ev.Waitlisted())
		// ANCHOR_END: debug
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrNotRegistered):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, ErrEmptyName):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// ANCHOR_END: cancel

func (h *handler) show(w http.ResponseWriter, r *http.Request) {
	ev, ok := h.store.Get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventView{Seats: ev.Seats(), Confirmed: ev.Confirmed(), Waitlist: ev.Waitlisted()})
}
