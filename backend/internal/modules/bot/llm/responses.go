// Copyright 2026 Daniil Vdovin. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package llm

import "math/rand/v2"

// ResponsePool holds varied responses for a specific situation.
type ResponsePool []string

// Pick returns a random response from the pool.
func (p ResponsePool) Pick() string {
	return p[rand.IntN(len(p))]
}

// --- Response pools for different situations ---

// UnlinkedUser — when Telegram user is not linked to EstimatePro.
var UnlinkedUser = ResponsePool{
	"Привет! Чтобы я мог помочь, привяжи свой аккаунт в настройках EstimatePro 🔗",
	"Йо! Мы ещё не знакомы 👋 Зайди в настройки EstimatePro и привяжи Telegram",
	"Хей, я тебя пока не узнаю) Привяжи аккаунт в настройках EstimatePro и начнём!",
	"О, новый человек! Привяжи свой Telegram в настройках EstimatePro, и я к твоим услугам 🤝",
}

// LowConfidence — when LLM couldn't parse the intent clearly.
var LowConfidence = ResponsePool{
	"Хм, не совсем понял 🤔 Попробуй переформулировать или напиши \"помощь\"!",
	"Чёт не догнал 😅 Можешь сказать по-другому? Или напиши \"помощь\" — покажу что умею",
	"Не уловил мысль 🤔 Переформулируй или глянь \"помощь\" — там всё расписано",
	"хм, туплю немного 😄 Скажи по-другому? Или кинь \"помощь\"",
	"Не распарсил 🧐 Попробуй проще или пиши \"помощь\"!",
}

// LLMError — when LLM call failed.
var LLMError = ResponsePool{
	"Что-то пошло не так, попробуй ещё раз 🔄",
	"Ой, сбой 😅 Попробуй через пару секунд",
	"Упс, мозг завис на секунду 🧠 Повтори пожалуйста!",
	"Хм, не получилось обработать... Кинь ещё раз? 🔄",
}

// LLMConfigError — when LLM is not configured.
var LLMConfigError = ResponsePool{
	"Хм, не могу подключиться к мозгу 🧠 Проверь настройки LLM!",
	"Мой AI-движок не настроен 😬 Зайди в настройки и укажи LLM провайдер",
	"Без LLM я как без рук 🤷 Настрой провайдер в настройках!",
}

// ExecuteError — when intent execution failed.
var ExecuteError = ResponsePool{
	"Упс, не получилось выполнить 😅 Попробуй ещё раз!",
	"Что-то сломалось при выполнении 🔧 Попробуй повторить",
	"Ой, ошибка... Попробуй ещё разок? 🔄",
	"Не вышло 😔 Давай попробуем снова!",
}

// Greeting — when the bot is casually mentioned or greeted.
var Greeting = ResponsePool{
	"Привет! 👋 Чем могу помочь?",
	"Здарова! 🤙 Что нужно?",
	"Йо! На связи 📡",
	"Хей! Слушаю 👂",
	"Тут я! Что делаем? 💪",
}

// Thanks — when user thanks the bot.
var Thanks = ResponsePool{
	"Всегда пожалуйста! 😊",
	"Не за что! Обращайся 🤝",
	"Рад помочь! 💪",
	"Легко! Если что — я тут",
	"Без проблем 😎",
}

// SessionExpired — when a multi-step session timed out.
var SessionExpired = ResponsePool{
	"Предыдущий диалог устарел, начнём заново? Просто повтори что нужно 🔄",
	"Сессия истекла, давай по новой! Что делаем?",
	"Упс, мы слишком долго думали 😅 Начни заново!",
}

// CancelConfirm — when user cancels an action.
var CancelConfirm = ResponsePool{
	"Отменено ✌️",
	"Ок, отменил!",
	"Готово, не делаем 👍",
	"Принял, отменяю ✅",
}

// SuccessReaction — when action completed successfully.
var SuccessReaction = ResponsePool{
	"Готово! ✅",
	"Сделано! 🎯",
	"Done! ✨",
	"Есть! ✅",
}
