package usage

import "strings"

type ModelPricing struct {
	InputPerToken  float64 // USD per token
	OutputPerToken float64 // USD per token
}

// per-million-token to per-token
func ppm(usdPerMillion float64) float64 {
	return usdPerMillion / 1_000_000
}

var defaultPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4o":      {InputPerToken: ppm(2.50), OutputPerToken: ppm(10.00)},
	"gpt-4o-mini": {InputPerToken: ppm(0.15), OutputPerToken: ppm(0.60)},
	"gpt-4-turbo": {InputPerToken: ppm(10.00), OutputPerToken: ppm(30.00)},
	"o1":          {InputPerToken: ppm(15.00), OutputPerToken: ppm(60.00)},
	"o1-mini":     {InputPerToken: ppm(3.00), OutputPerToken: ppm(12.00)},

	// Anthropic
	"claude-sonnet-4-20250514":   {InputPerToken: ppm(3.00), OutputPerToken: ppm(15.00)},
	"claude-3-5-sonnet-20241022": {InputPerToken: ppm(3.00), OutputPerToken: ppm(15.00)},
	"claude-3-haiku-20240307":    {InputPerToken: ppm(0.25), OutputPerToken: ppm(1.25)},
	"claude-3-opus-20240229":     {InputPerToken: ppm(15.00), OutputPerToken: ppm(75.00)},

	// DeepSeek
	"deepseek-chat":     {InputPerToken: ppm(0.27), OutputPerToken: ppm(1.10)},
	"deepseek-reasoner": {InputPerToken: ppm(0.55), OutputPerToken: ppm(2.19)},

	// Moonshot (Kimi)
	"moonshot-v1-8k":   {InputPerToken: ppm(1.00), OutputPerToken: ppm(1.00)},
	"moonshot-v1-32k":  {InputPerToken: ppm(2.00), OutputPerToken: ppm(2.00)},
	"moonshot-v1-128k": {InputPerToken: ppm(5.00), OutputPerToken: ppm(5.00)},

	// Zhipu (GLM)
	"glm-4":       {InputPerToken: ppm(1.00), OutputPerToken: ppm(1.00)},
	"glm-4-flash": {InputPerToken: ppm(0.10), OutputPerToken: ppm(0.10)},
	"glm-4-plus":  {InputPerToken: ppm(5.00), OutputPerToken: ppm(5.00)},

	// MiniMax
	"MiniMax-Text-01": {InputPerToken: ppm(1.00), OutputPerToken: ppm(1.00)},
	"abab6.5s-chat":   {InputPerToken: ppm(1.00), OutputPerToken: ppm(1.00)},

	// DashScope (Qwen / Tongyi)
	"qwen-plus":    {InputPerToken: ppm(0.80), OutputPerToken: ppm(2.00)},
	"qwen-turbo":   {InputPerToken: ppm(0.30), OutputPerToken: ppm(0.60)},
	"qwen-max":     {InputPerToken: ppm(2.00), OutputPerToken: ppm(6.00)},
	"qwen-long":    {InputPerToken: ppm(0.50), OutputPerToken: ppm(2.00)},
	"qwen-vl-plus": {InputPerToken: ppm(0.80), OutputPerToken: ppm(2.00)},
}

var zeroPricing = ModelPricing{}

func GetPricing(model string) ModelPricing {
	if p, ok := defaultPricing[model]; ok {
		return p
	}
	lower := strings.ToLower(model)
	for k, v := range defaultPricing {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return zeroPricing
}
