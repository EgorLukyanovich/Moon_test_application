package models

type ErrorResponse struct {
	Error struct {
		Code string `json:"code"`
		Text string `json:"text"`
	} `json:"error"`
}
