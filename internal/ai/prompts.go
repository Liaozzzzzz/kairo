package ai

import (
	_ "embed"
	"strings"
)

//go:embed prompts/category_english.txt
var promptEnglish string

//go:embed prompts/category_speech.txt
var promptSpeech string

//go:embed prompts/category_emotion.txt
var promptEmotion string

//go:embed prompts/category_experience.txt
var promptExperience string

func GetCategoryPrompts() map[string]string {
	return map[string]string{
		"英语口语": strings.TrimSpace(promptEnglish),
		"演讲":   strings.TrimSpace(promptSpeech),
		"情感关系": strings.TrimSpace(promptEmotion),
		"经验分享": strings.TrimSpace(promptExperience),
	}
}
