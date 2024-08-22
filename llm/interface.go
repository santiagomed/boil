package llm

type LlmClient interface {
	GetCompletion(prompt, responseType string) (string, error)
}
