package llm

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*AnthropicProvider)(nil),
		(*DeepSeekProvider)(nil),
		(*GeminiProvider)(nil),
		(*LLMRouter)(nil),
		(*OllamaProvider)(nil),
		(*OpenAIProvider)(nil),
		(*QwenProvider)(nil),
	}
}
