## ADDED Requirements

### Requirement: Chat button on entry cards
The system SHALL display a chat icon (robot/AI icon) on each entry card that opens a chat panel for that article.

#### Scenario: Open chat panel
- **WHEN** user taps the chat icon on an entry card
- **THEN** a chat panel opens with the article title and an empty conversation

### Requirement: Chat grounded in article content
The system SHALL send the article's raw_content as context in the system prompt. The LLM SHALL be instructed to answer ONLY based on the article content. If the article does not contain the answer, the LLM SHALL state this clearly.

#### Scenario: Question answerable from article
- **WHEN** user asks "What benchmarks were used?" and the article discusses specific benchmarks
- **THEN** the AI responds with benchmark details from the article

#### Scenario: Question not in article
- **WHEN** user asks "What is the author's Twitter handle?" and the article does not mention it
- **THEN** the AI responds that this information is not in the article

### Requirement: Streaming chat responses
The system SHALL stream chat responses token-by-token to the frontend. Tokens SHALL appear as they are received from OpenRouter.

#### Scenario: Streaming response display
- **WHEN** user sends a question and the LLM generates a response
- **THEN** the response text appears incrementally in the chat panel as tokens arrive

### Requirement: Conversation history in chat session
The system SHALL maintain the conversation history within a chat session. Each message sent includes the full conversation history so the LLM can reference previous questions and answers.

#### Scenario: Follow-up question
- **WHEN** user asks "What's the main argument?" then follows up with "Can you elaborate on the second point?"
- **THEN** the AI references its previous answer and provides more detail on the second point

### Requirement: Ephemeral chat sessions
Chat sessions SHALL NOT be persisted to the database. Closing the chat panel ends the session. The conversation history lives only in the browser.

#### Scenario: Close and reopen chat
- **WHEN** user has a 5-message conversation, closes the chat panel, then reopens it
- **THEN** the chat panel shows an empty conversation

### Requirement: Chat uses configured model
The system SHALL use the same OpenRouter model configured in settings for chat. The user MAY use a different model than the one used for summarization.

#### Scenario: Chat with configured model
- **WHEN** user opens a chat and the configured model is "anthropic/claude-sonnet-4"
- **THEN** the chat requests are sent to that model via OpenRouter

### Requirement: Mobile chat layout
On mobile viewports, the chat panel SHALL display as a full-screen overlay with the conversation and input field.

#### Scenario: Mobile chat view
- **WHEN** user opens the chat panel on a 375px-wide screen
- **THEN** the chat takes the full screen with a close button, scrollable messages, and a fixed input field at the bottom
