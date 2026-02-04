package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v2"
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
	imagesDir    = "images"
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
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		fmt.Printf("Error creating images directory: %v\n", err)
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
			fmt.Println("Assistant: Generating and saving image...")
			imagePath, err := generateImage(ctx, client, userInput)
			if err != nil {
				fmt.Printf("Error generating image: %v\n", err)
				continue
			}
			fmt.Printf("Assistant: Image saved to: %s\n\n", imagePath)
			openFile(imagePath)
			conv.addMessage("assistant", fmt.Sprintf("Generated image: %s", imagePath))
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

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    defaultModel,
		Messages: messages,
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
	resp, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          openai.ImageModelDallE3,
		Size:           "1024x1024",
		ResponseFormat: "b64_json",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create image: %w", err)
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("no image data in response")
	}

	imgData, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %w", err)
	}

	fileName := fmt.Sprintf("img_%d.png", time.Now().Unix())
	filePath := filepath.Join(imagesDir, fileName)

	err = os.WriteFile(filePath, imgData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write image to file: %w", err)
	}

	return filePath, nil
}

func openFile(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = exec.Command("xdg-open", path)
	}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
	}
}
