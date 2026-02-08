# OpenAI CLI Chat

A simple command-line chat application that interacts with OpenAI's API and stores conversations in XML format.

## Features

- Interactive CLI chat interface with OpenAI
- Conversations stored as XML files in the `chats` directory
- System prompt included in every conversation
- Timestamps for all messages
- MIT licensed

## Prerequisites

- Go 1.21 or higher
- OpenAI API key

## Installation

1. Clone or download this repository

2. Install dependencies:
```bash
go mod download
```

3. Set your OpenAI API key as an environment variable:
```bash
export OPENAI_KEY='your-api-key-here'
```

## Usage

Run the application:
```bash
go run main.go
```

Or build and run:
```bash
go build -o chat
./chat
```

### Chat Commands

- Type your message and press Enter to send
- Type `exit` or `quit` to end the conversation and save

## Conversation Storage

All conversations are automatically saved to the `chats` directory as XML files when you exit. Each file is named with a unique timestamp ID (e.g., `chat_1738598400.xml`).

### XML Format

```xml
<conversation id="chat_1738598400" created_at="2026-02-03T10:00:00Z">
  <messages>
    <message role="system" timestamp="2026-02-03T10:00:00Z">
      <content>You are a helpful assistant. Provide clear, concise, and accurate responses.</content>
    </message>
    <message role="user" timestamp="2026-02-03T10:01:00Z">
      <content>Hello!</content>
    </message>
    <message role="assistant" timestamp="2026-02-03T10:01:05Z">
      <content>Hello! How can I help you today?</content>
    </message>
  </messages>
</conversation>
```

## Configuration

You can modify the following constants in `main.go`:

- `systemPrompt`: The initial system prompt sent to the AI
- `defaultModel`: The OpenAI model to use (default: "gpt-5")
- `chatsDir`: Directory where conversations are saved (default: "chats")

## License

MIT License - see LICENSE file for details
