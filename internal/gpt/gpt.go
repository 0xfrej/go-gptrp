package gpt

import (
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"gptrp/internal/config"
	"io"
	"log"
)

const NarratorPersonalityDefinition = "You're a narrator %s."

type StreamConsumer func(string)

type Context struct {
	isDungeon   bool
	ctxMessages []openai.ChatCompletionMessage
	gpt         *GPT
}

type Settings struct {
	Scenario         string
	Model            string
	StoreGptMessages bool
	MaxTokens        int
}

type GPT struct {
	client   *openai.Client
	settings Settings
	cfg      *config.Config
	ctx      *Context
	scenario config.Scenario
}

func (g *GPT) GetScenario() config.Scenario {
	return g.scenario
}

func NewGpt(cfg *config.Config, settings Settings) (GPT, error) {
	scenario, err := cfg.GetScenario(settings.Scenario)
	if err != nil {
		return GPT{}, err
	}
	return GPT{
			client:   openai.NewClient(cfg.OpenAI.ApiKey),
			settings: settings,
			cfg:      cfg,
			scenario: scenario,
			ctx: &Context{
				isDungeon:   false,
				ctxMessages: createWorldMessages(&scenario),
			},
		},
		nil
}

func (g *GPT) NewContext(
	isDungeon bool,
	extraBuilding string,
) *GPT {
	messages := append(createWorldMessages(&g.scenario), createDungeonMessages(&g.scenario, extraBuilding)...)
	g.ctx = &Context{
		isDungeon:   isDungeon,
		ctxMessages: messages,
	}

	return g
}

func (g *GPT) prepareRequest() openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:    g.settings.Model,
		Messages: g.ctx.ctxMessages,
	}

	if g.settings.MaxTokens > 0 {
		req.MaxTokens = g.settings.MaxTokens
	}

	return req
}

func (g *GPT) WasLastMessageFromAssistant() bool {
	return g.wasLastMessageFrom(openai.ChatMessageRoleAssistant)
}

func (g *GPT) WasLastMessageFromUser() bool {
	return g.wasLastMessageFrom(openai.ChatMessageRoleUser)
}

func (g *GPT) wasLastMessageFrom(role string) bool {
	return g.ctx.ctxMessages[len(g.ctx.ctxMessages)-1].Role == role
}

func (g *GPT) RedoLastMessage() *GPT {
	if g.WasLastMessageFromAssistant() {
		g.RemoveLastMessage()
	}
	return g
}

func (g *GPT) Undo() *GPT {
	if g.WasLastMessageFromAssistant() {
		g.RemoveLastMessage()
	}
	g.RemoveLastMessage()
	return g
}

func (g *GPT) GetContextMessages() []openai.ChatCompletionMessage {
	return g.ctx.ctxMessages
}

func (g *GPT) ChatCompletion() string {
	resp, err := g.client.CreateChatCompletion(
		context.Background(),
		g.prepareRequest(),
	)
	if err != nil {
		log.Printf("Error creating chat completion: %v\n", err)
		return ""
	}

	msg := resp.Choices[0].Message.Content
	if g.settings.StoreGptMessages {
		g.ctx.ctxMessages = append(g.ctx.ctxMessages, createGptMessage(msg))
	}
	return msg
}

func (g *GPT) ChatCompletionStream(fn StreamConsumer) {
	stream, err := g.client.CreateChatCompletionStream(
		context.Background(),
		g.prepareRequest(),
	)
	if err != nil {
		log.Printf("Error creating chat completion: %v\n", err)
		for _, msg := range g.ctx.ctxMessages {
			fmt.Printf("%s: %s\n", msg.Role, msg.Content)
		}
		return
	}
	defer stream.Close()

	msg := ""
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			log.Printf("\nStream error: %v\n", err)
			return
		}
		msg += response.Choices[0].Delta.Content
		fn(response.Choices[0].Delta.Content)
	}

	if g.settings.StoreGptMessages {
		g.ctx.ctxMessages = append(g.ctx.ctxMessages, createGptMessage(msg))
	}
}

func (g *GPT) AddMessage(message string) *GPT {
	g.ctx.ctxMessages = append(g.ctx.ctxMessages, createUserMessage(message))
	return g
}

func (g *GPT) RemoveLastMessage() *GPT {
	if len(g.ctx.ctxMessages) > 0 {
		g.ctx.ctxMessages = g.ctx.ctxMessages[:len(g.ctx.ctxMessages)-1]
	}
	return g
}

func createWorldMessages(scenario *config.Scenario) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage
	if len(scenario.NarratorPersonality) > 0 {
		messages = append(messages, createBuildingMessage(fmt.Sprintf(NarratorPersonalityDefinition, scenario.NarratorPersonality)))
	}
	if len(scenario.WorldBuilding) > 0 {
		messages = append(messages, createBuildingMessage(scenario.WorldBuilding))
	}

	return messages
}

func createDungeonMessages(scenario *config.Scenario, extraBuilding string) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage

	if len(scenario.DungeonRoomBuilding) > 0 {
		messages = append(messages, createBuildingMessage(scenario.DungeonRoomBuilding))
	}
	if len(extraBuilding) > 0 {
		messages = append(messages, createBuildingMessage(extraBuilding))
	}

	return messages
}

func createBuildingMessage(msg string) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: msg,
	}
}

func createGptMessage(msg string) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: msg,
	}
}

func createUserMessage(msg string) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
	}
}
