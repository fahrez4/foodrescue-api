package models

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func BuildResponse(success bool, message string, data interface{}) Response {
	return Response{
		Success: success,
		Message: message,
		Data:    data,
	}
}

func BuildErrorResponse(message string) Response {
	return Response{
		Success: false,
		Message: message,
		Data:    nil,
	}
}
