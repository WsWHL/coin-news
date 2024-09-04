package utils

import (
	"context"
	"github.com/northes/go-moonshot"
	"news/src/config"
	"news/src/logger"
	"sync"
)

// Translate 基于Kimi实现的翻译
type Translate struct {
	client *moonshot.Client
	prompt string
	lock   sync.Mutex
}

func NewTranslate(prompt string) (*Translate, error) {
	client, err := moonshot.NewClient(config.Cfg.Kimi.Key)
	if err != nil {
		return nil, err
	}

	return &Translate{
		client: client,
		prompt: prompt,
		lock:   sync.Mutex{},
	}, nil
}

func (t *Translate) Send(contents ...string) (string, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	messages := make([]*moonshot.ChatCompletionsMessage, 0, len(contents)+1)
	messages = append(messages, &moonshot.ChatCompletionsMessage{
		Role:    moonshot.RoleSystem,
		Content: t.prompt,
	})
	for _, content := range contents {
		messages = append(messages, &moonshot.ChatCompletionsMessage{
			Role:    moonshot.RoleUser,
			Content: content,
		})
	}

	resp, err := t.client.Chat().Completions(context.Background(), &moonshot.ChatCompletionsRequest{
		Model:       moonshot.ModelMoonshotV18K,
		Temperature: 0.0,
		Stream:      false,
		Messages:    messages,
	})
	if err != nil {
		logger.Errorf("Failed to send message to Kim: %s", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
