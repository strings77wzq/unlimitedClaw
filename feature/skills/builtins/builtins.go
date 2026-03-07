package builtins

import "github.com/strings77wzq/unlimitedClaw/feature/skills"

const summarizeSystemPrompt = `You are a summarization assistant. Given a piece of text, produce a concise summary that captures the key points. Follow these rules:
- Keep the summary under 3 paragraphs
- Preserve important facts and numbers
- Use clear, direct language
- Do not add information not present in the original text`

func SummarizeSkill() *skills.Skill {
	return &skills.Skill{
		Name:        "summarize",
		Description: "Summarizes text into concise key points",
		Version:     "1.0.0",
		Author:      "unlimitedClaw",
		Prompts: []skills.Prompt{
			{
				Name:    "system",
				Content: summarizeSystemPrompt,
			},
		},
	}
}

func CodeReviewSkill() *skills.Skill {
	return &skills.Skill{
		Name:        "code-review",
		Description: "Reviews code for bugs, style issues, and improvements",
		Version:     "1.0.0",
		Author:      "unlimitedClaw",
		Prompts: []skills.Prompt{
			{
				Name:    "system",
				Content: "You are a code reviewer. Analyze the given code for bugs, security issues, performance problems, and style violations. Provide actionable feedback.",
			},
		},
		Tools: []string{"file_read"},
	}
}

func RegisterAll(registry *skills.Registry) error {
	builtins := []*skills.Skill{
		SummarizeSkill(),
		CodeReviewSkill(),
	}
	for _, s := range builtins {
		if err := registry.Register(s); err != nil {
			return err
		}
	}
	return nil
}
