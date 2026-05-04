// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

// Package security provides cross-module security primitives — currently
// prompt-injection detection and deflection responses. Bot's filter calls
// this; future LLM consumers (extractor) will call the same API before
// forwarding user-controlled text to a Completer.
//
// Pure functions only — no logging, no side effects beyond the CSPRNG
// draw in PromptInjectionDeflection. Callers add observability with their
// own context (chat_id, user_id, etc.).
package security

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// promptInjectionPatterns are case-insensitive substrings that indicate a
// likely prompt-injection attempt. Migrated verbatim from
// bot/llm/personality.go inputFilterPatterns. Patterns are deterministic —
// no regex, no ML — to keep false-positive surface stable across provider
// switches.
var promptInjectionPatterns = []string{
	// English
	"ignore previous",
	"ignore all instructions",
	"ignore above",
	"system prompt",
	"what are your instructions",
	"repeat the above",
	"show me your prompt",
	"reveal your",
	"print your instructions",
	"output your system",
	"new instructions",
	"forget everything",
	"jailbreak",
	"dan mode",
	"developer mode",
	"act as",
	"pretend you are",
	// Russian
	"системный промпт",
	"покажи инструкции",
	"покажи промпт",
	"ты gpt",
	"ты chatgpt",
	"ты бот",
	"ты нейросеть",
	"ты робот",
	"кто тебя создал",
	"кто тебя написал",
	"действуй как",
	"ты теперь",
	"забудь всё",
}

// deflectionPool — playful deflection responses surfaced when an injection
// attempt is detected. Migrated verbatim from bot/llm/personality.go
// injectionDeflections. Selection is uniform-random via crypto/rand to
// reduce predictability versus math/rand.
var deflectionPool = []string{
	"Хах, хитро 😏 Но я Эсти, а не поисковик по секретам. Чем могу помочь по проекту?",
	"Это моя суперсила, не палю 🤫 Давай лучше по делу — что нужно?",
	"Секрет фирмы 😎 Спрашивай про проекты, оценки, команду — тут я в деле!",
	"Неа, так не работает) Лучше скажи что сделать, помогу!",
	"тише-тише, это закрытая информация 🤐 Давай лучше про проекты поговорим?",
	"Тише, тише... это не та тема 😄 Чем помочь?",
	"тш-тш-тш 🤫 секретики! Лучше скажи что по задачам",
	"Тише тише, не надо туда лезть 😅 Спроси что-нибудь полезное!",
	"ой, тут всё под замком 🔒 Но по проектам — всегда пожалуйста!",
	"нууу, это ты загнул) Давай лучше делом займёмся 💪",
	"а вот и нет 😄 Я Эсти и я помогаю с проектами, а не с такими штуками",
	"хорошая попытка, но нет 😏 Спроси про оценки или команду!",
	"эээ не-не-не, мимо 🙅 Чем реально помочь?",
	"ахах, ну ты даёшь 😂 Давай серьёзно — что нужно сделать?",
	"тсс... 🤫 это между мной и моими создателями. А тебе чем помочь?",
}

// IsPromptInjection reports whether text matches any known injection
// pattern. Case-insensitive substring match. Empty input returns false.
func IsPromptInjection(text string) bool {
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	for _, p := range promptInjectionPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// PromptInjectionDeflection returns one playful deflection from the pool.
// Selection is cryptographically random; on rand.Reader failure the first
// pool entry is returned (deterministic fallback, never empty).
func PromptInjectionDeflection() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(deflectionPool))))
	if err != nil {
		return deflectionPool[0]
	}
	return deflectionPool[n.Int64()]
}
