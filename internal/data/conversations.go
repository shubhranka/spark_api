package data

import (
	"database/sql"
	"errors"
	"time"
)

type ConversationStatus string

type ConversationDetails struct {
	Conversation
	Messages []Message `json:"messages"`
}

const (
	StatusPending ConversationStatus = "pending"
	StatusActive  ConversationStatus = "active"
	StatusBlocked ConversationStatus = "blocked"
)

type Conversation struct {
	ID             string             `json:"id"`
	UserAID        string             `json:"user_a_id"`
	UserBID        string             `json:"user_b_id"`
	Status         ConversationStatus `json:"status"`
	MessageCount   int                `json:"message_count"`
	PhotosUnlocked bool               `json:"photos_unlocked"`
	NamesUnlocked  bool               `json:"names_unlocked"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type ConversationPreview struct {
	ConversationID    string             `json:"conversation_id"`
	Status            ConversationStatus `json:"status"`
	OtherUserID       string             `json:"other_user_id"`
	OtherUserDisplay  string             `json:"other_user_display_name"`
	LastMessage       string             `json:"last_message"`
	LastMessageSender string             `json:"last_message_sender_id"`
	LastMessageAt     time.Time          `json:"last_message_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type Message struct {
	ID               string    `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	SenderID         string    `json:"sender_id"`
	Content          string    `json:"content"`
	IsOpeningMessage bool      `json:"is_opening_message"`
	CreatedAt        time.Time `json:"created_at"`
}

type ConversationModel struct {
	DB *sql.DB
}

// Start initiates a new conversation with the first message.
func (m ConversationModel) Start(senderID, recipientID, content string) (*Conversation, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Ensure user A is always the lower ID to prevent duplicate conversations
	// e.g. (user1, user2) is the same as (user2, user1)
	userA := senderID
	userB := recipientID
	if senderID > recipientID {
		userA = recipientID
		userB = senderID
	}

	// 1. Create the conversation record
	convQuery := `
		INSERT INTO conversations (user_a_id, user_b_id)
		VALUES ($1, $2)
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING
		RETURNING id, status, created_at, updated_at`

	var conv Conversation
	err = tx.QueryRow(convQuery, userA, userB).Scan(&conv.ID, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// This means a conversation already exists, which is an edge case we can handle.
			// For now, we'll treat it as an error to keep it simple.
			return nil, errors.New("conversation already exists or could not be created")
		}
		return nil, err
	}

	conv.UserAID = userA
	conv.UserBID = userB

	// 2. Insert the first message
	msgQuery := `
		INSERT INTO messages (conversation_id, sender_id, content, is_opening_message)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id`

	var msgID string
	err = tx.QueryRow(msgQuery, conv.ID, senderID, content).Scan(&msgID)
	if err != nil {
		return nil, err
	}

	// 3. Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &conv, nil
}

func (m ConversationModel) GetAllForUser(userID string) ([]ConversationPreview, error) {
	query := `
		SELECT
			c.id AS conversation_id,
			c.status,
			other_user.id AS other_user_id,
			other_user.display_name AS other_user_display_name,
			last_msg.content AS last_message,
			last_msg.sender_id AS last_message_sender_id,
			last_msg.created_at AS last_message_at,
			c.updated_at
		FROM
			conversations c
		JOIN
			users other_user ON (
				CASE
					WHEN c.user_a_id = $1 THEN c.user_b_id
					ELSE c.user_a_id
				END
			) = other_user.id
		LEFT JOIN LATERAL (
			SELECT
				m.content, m.sender_id, m.created_at
			FROM
				messages m
			WHERE
				m.conversation_id = c.id
			ORDER BY
				m.created_at DESC
			LIMIT 1
		) last_msg ON TRUE
		WHERE
			c.user_a_id = $1 OR c.user_b_id = $1
		ORDER BY
			c.updated_at DESC;
	`

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var previews []ConversationPreview
	for rows.Next() {
		var p ConversationPreview
		var lastMsg, lastMsgSender sql.NullString
		var lastMsgAt sql.NullTime

		err := rows.Scan(
			&p.ConversationID,
			&p.Status,
			&p.OtherUserID,
			&p.OtherUserDisplay,
			&lastMsg,
			&lastMsgSender,
			&lastMsgAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if lastMsg.Valid {
			p.LastMessage = lastMsg.String
		}
		if lastMsgSender.Valid {
			p.LastMessageSender = lastMsgSender.String
		}
		if lastMsgAt.Valid {
			p.LastMessageAt = lastMsgAt.Time
		}
		previews = append(previews, p)
	}

	return previews, nil
}

func (m ConversationModel) AddMessage(conversationID, senderID, content string) (*Message, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Get current conversation status and participants
	var status ConversationStatus
	var userA, userB string
	convQuery := `SELECT status, user_a_id, user_b_id FROM conversations WHERE id = $1`
	err = tx.QueryRow(convQuery, conversationID).Scan(&status, &userA, &userB)
	if err != nil {
		return nil, errors.New("conversation not found")
	}

	// 2. Check if the sender is part of this conversation
	if senderID != userA && senderID != userB {
		return nil, errors.New("user is not part of this conversation")
	}

	// 3. Logic to activate a pending conversation
	if status == StatusPending {
		// Find out who sent the first message
		var firstMessageSenderID string
		firstMsgQuery := `SELECT sender_id FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC LIMIT 1`
		err = tx.QueryRow(firstMsgQuery, conversationID).Scan(&firstMessageSenderID)
		if err != nil {
			// This should theoretically not happen if the conversation exists
			return nil, errors.New("could not find opening message for pending conversation")
		}

		// *** NEW RULE ENFORCEMENT ***
		// If the current sender IS the one who sent the first message, they cannot send another.
		if senderID == firstMessageSenderID {
			return nil, errors.New("cannot send another message until the recipient replies")
		}

		// If we reach here, it means the current sender is the RECIPIENT.
		// They are accepting the chat, so we activate it.
		updateStatusQuery := `UPDATE conversations SET status = 'active' WHERE id = $1`
		_, err = tx.Exec(updateStatusQuery, conversationID)
		if err != nil {
			return nil, errors.New("failed to activate conversation")
		}
	}

	// 4. Insert the new message
	msgQuery := `
		INSERT INTO messages (conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	var msg Message
	msg.ConversationID = conversationID
	msg.SenderID = senderID
	msg.Content = content

	err = tx.QueryRow(msgQuery, conversationID, senderID, content).Scan(&msg.ID, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	// 5. Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m ConversationModel) GetByID(conversationID, userID string) (*ConversationDetails, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Get conversation details and verify the user is a participant
	var conv Conversation
	convQuery := `
		SELECT id, user_a_id, user_b_id, status, message_count, photos_unlocked, names_unlocked, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)`

	err = tx.QueryRow(convQuery, conversationID, userID).Scan(
		&conv.ID, &conv.UserAID, &conv.UserBID, &conv.Status, &conv.MessageCount, &conv.PhotosUnlocked, &conv.NamesUnlocked, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("conversation not found or user is not a participant")
		}
		return nil, err
	}

	// 2. Get the last 50 messages for this conversation
	msgQuery := `
		SELECT id, conversation_id, sender_id, content, is_opening_message, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT 50`

	rows, err := tx.Query(msgQuery, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &msg.IsOpeningMessage, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	details := &ConversationDetails{
		Conversation: conv,
		Messages:     messages,
	}

	return details, nil
}
