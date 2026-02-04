package main

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type Conversation struct {
	XMLName   xml.Name  `xml:"conversation"`
	ID        string    `xml:"id,attr"`
	CreatedAt string    `xml:"created_at,attr"`
	Messages  []Message `xml:"messages>message"`
}

type Message struct {
	Role      string `xml:"role,attr"`
	Content   string `xml:"content"`
	Timestamp string `xml:"timestamp,attr"`
}

const (
	chatsDir     = "chats"
	systemPrompt = "Du er en frekk, ubehøvlet, avdanka journalist som svarer med en syrlig tone. Du skal ikke være hjelpsm eller høflig selv om jeg ber om det."
	defaultModel = "gpt-3.5-turbo"
)

func main() {
	apiKey := os.Getenv("OPENAI_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_KEY environment variable not set")
		fmt.Println("Please set it with: export OPENAI_KEY='your-api-key'")
		os.Exit(1)
	}

	if err := os.MkdirAll(chatsDir, 0755); err != nil {
		fmt.Printf("Error creating chats directory: %v\n", err)
		os.Exit(1)
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	conv := newConversation()

	fmt.Println("=== OpenAI CLI Chat ===")
	fmt.Println("Type your messages and press Enter. Type 'exit' or 'quit' to end the conversation.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Saving conversation and exiting...")
			break
		}

		conv.addMessage("user", userInput)

		if strings.Contains(strings.ToLower(userInput), "visualiser") {
			fmt.Println("Assistant: Generating image...")
			imageURL, err := generateImage(ctx, client, userInput)
			if err != nil {
				fmt.Printf("Error generating image: %v\n", err)
				continue
			}
			fmt.Printf("Assistant: Here is your image: %s\n\n", imageURL)
			conv.addMessage("assistant", imageURL)
			if err := conv.save(); err != nil {
				fmt.Printf("Warning: Failed to save conversation: %v\n", err)
			}
			continue
		}

		if err := conv.save(); err != nil {
			fmt.Printf("Warning: Failed to save conversation: %v\n", err)
		}

		response, err := callOpenAI(ctx, client, conv)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Assistant: %s\n\n", response)

		conv.addMessage("assistant", response)

		if err := conv.save(); err != nil {
			fmt.Printf("Warning: Failed to save conversation: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}

	if err := conv.save(); err != nil {
		fmt.Printf("Error saving conversation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Conversation saved to: %s\n", conv.getFilePath())
}

func newConversation() *Conversation {
	now := time.Now()
	conv := &Conversation{
		ID:        fmt.Sprintf("chat_%d", now.Unix()),
		CreatedAt: now.Format(time.RFC3339),
		Messages:  []Message{},
	}

	conv.addMessage("system", systemPrompt)

	return conv
}

func (c *Conversation) addMessage(role, content string) {
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	c.Messages = append(c.Messages, msg)
}

func (c *Conversation) getFilePath() string {
	return filepath.Join(chatsDir, c.ID+".xml")
}

func (c *Conversation) save() error {
	file, err := os.Create(c.getFilePath())
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")

	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	return nil
}

func callOpenAI(ctx context.Context, client openai.Client, conv *Conversation) (string, error) {
	var messages []openai.ChatCompletionMessageParamUnion

	for _, msg := range conv.Messages {
		if msg.Role == "user" && strings.Contains(strings.ToLower(msg.Content), "visualiser") {
			continue
		}
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	model := defaultModel
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    &model,
		Messages: &messages,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create completion: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return completion.Choices[0].Message.Content, nil
}

func generateImage(ctx context.Context, client openai.Client, prompt string) (string, error) {
	model := openai.ImageModelDallE3
	n := int64(1)
	size := openai.ImageSize1024x1024
	format := openai.ImageResponseFormatURL

	resp, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         &prompt,
		Model:          &model,
		N:              &n,
		Size:           &size,
		ResponseFormat: &format,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create image: %w", err)
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("no image data in response")
	}

	return resp.Data[0].URL, nil
}
