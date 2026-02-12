package json_resp

import (
	"encoding/json"
	"log"
	"net/http"

	models "github.com/egor_lukyanovich/moon_test_application/internal/models"
)

func RespondError(w http.ResponseWriter, code int, errCode, msg string) {
	if code >= 500 {
		log.Println("Server error:", msg)
	}

	res := models.ErrorResponse{}
	res.Error.Code = errCode
	res.Error.Text = msg

	RespondJSON(w, code, res)
}

func RespondJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("marshal json error:", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log.Println("write response failed:", err)
	}

}
