package api

// APIError represents a structured API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e APIError) Error() string {
	return e.Message
}

// Common API errors with codes
var (
	ErrNotFound = APIError{
		Code:    "NOT_FOUND",
		Message: "资源不存在",
	}
	ErrParseFailed = APIError{
		Code:    "PARSE_FAILED",
		Message: "解析 Beancount 文件失败",
	}
	ErrRefreshFailed = APIError{
		Code:    "REFRESH_FAILED",
		Message: "刷新数据失败",
	}
	ErrRateLimited = APIError{
		Code:    "RATE_LIMITED",
		Message: "请求过于频繁，请稍后再试",
	}
	ErrInvalidRequest = APIError{
		Code:    "INVALID_REQUEST",
		Message: "请求参数无效",
	}
	ErrInternalError = APIError{
		Code:    "INTERNAL_ERROR",
		Message: "服务器内部错误",
	}
)

// NewAPIError creates a custom API error
func NewAPIError(code, message string) APIError {
	return APIError{Code: code, Message: message}
}
