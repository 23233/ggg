package pmb

import (
	"sync"
	"time"
)

type Message struct {
	UID       string          `json:"-"`
	Content   string          `json:"content,omitempty"`
	CreatedAt time.Time       `json:"created_at,omitempty"`
	Sequence  int             `json:"sequence,omitempty"`
	UA        map[string]bool `json:"-"`
}
type MessageQueue struct {
	msgMap  map[string][]Message
	counter map[string]int
	mu      sync.Mutex
}

func NewMessageQueue() *MessageQueue {
	mq := &MessageQueue{
		msgMap:  make(map[string][]Message),
		counter: make(map[string]int),
	}
	go mq.cleanUp()
	return mq
}

func (mq *MessageQueue) Put(uid, content string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	seq := mq.counter[uid] + 1
	mq.counter[uid] = seq
	message := Message{
		UID:       uid,
		Content:   content,
		UA:        make(map[string]bool),
		CreatedAt: time.Now(),
		Sequence:  seq,
	}
	mq.msgMap[uid] = append(mq.msgMap[uid], message)
}

func (mq *MessageQueue) Consume(uid, ua string) (Message, bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	messages, ok := mq.msgMap[uid]
	if !ok || len(messages) == 0 {
		return Message{}, false
	}

	for _, msg := range messages {
		if _, ok := msg.UA[ua]; !ok {
			msg.UA[ua] = true

			// Remove the consumed message after 30 seconds
			go func(uid string, sequence int) {
				time.Sleep(30 * time.Second)
				mq.mu.Lock()
				for i, msg := range mq.msgMap[uid] {
					if msg.Sequence == sequence {
						mq.msgMap[uid] = append(mq.msgMap[uid][:i], mq.msgMap[uid][i+1:]...)
						break
					}
				}
				mq.mu.Unlock()
			}(uid, msg.Sequence)

			return msg, true
		}
	}

	return Message{}, false
}
func (mq *MessageQueue) cleanUp() {
	for {
		time.Sleep(30 * time.Minute)
		mq.mu.Lock()
		now := time.Now()
		for uid, messages := range mq.msgMap {
			var newMessages []Message
			for _, msg := range messages {
				if now.Sub(msg.CreatedAt) < 48*time.Hour {
					newMessages = append(newMessages, msg)
				}
			}
			mq.msgMap[uid] = newMessages
		}
		mq.mu.Unlock()
	}
}
