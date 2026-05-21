package openai

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (r ChatCompletionRequest) Validate() error {
	if len(r.Messages) == 0 {
		return ErrMissingMessages
	}
	return nil
}

type validationError string

func (e validationError) Error() string {
	return string(e)
}

const ErrMissingMessages = validationError("messages is required")
