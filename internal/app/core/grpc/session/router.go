package session

import (
	"context"
	"fmt"
	"shared/proto"
	"shared/util"
)

// sendMessage sends a message with thread safety and context cancellation support
func (s *Session) sendMessage(ctx context.Context, msg *proto.ShellClientMessage) error {

	// Create a channel to receive the result of the send operation
	done := make(chan error, 1)

	// Add to wait group
	s.wg.Add(1)

	// Create a goroutine to send the message asynchronously
	go func() {
		defer s.wg.Done()
		done <- s.stream.Send(msg)
	}()

	// Wait for the send operation to complete or the context to be canceled
	select {
	case err := <-done:
		return err

	case <-ctx.Done():

		// Remove the pending response on message cancellation
		s.removePending(msg.Id)

		// Return the context error
		return ctx.Err()

	case <-s.closed:
		return s.closeErr
	}

}

// sendRequestWithContext sends a request to the peer and waits for a response
func (s *Session) sendRequestWithContext(ctx context.Context, msg *proto.ShellClientMessage) (*proto.ShellServerMessage, error) {

	// Generate a unique ID for the request
	msg.Id = newID()

	// Register a response channel for the request
	respCh := s.registerPending(msg.Id)

	// Send the request synchronously
	err := s.sendMessage(ctx, msg)

	// If the request failed, remove the pending channel and return the error
	if err != nil {
		return nil, err
	}

	select {

	case resp := <-respCh:

		// Return the response on success
		return resp, nil

	case <-ctx.Done():

		// Return the context error on cancellation
		return nil, ctx.Err()

	case <-s.closed:

		// If the client was closed, return the error notification
		return nil, s.closeErr

	}

}

// readLoop reads messages from the connection and delivers them to the message channel.
func (s *Session) readLoop() {

	// Decrement wait group counter on return
	defer s.wg.Done()

	for {

		// Decode incoming messages
		msg, err := s.stream.Recv()

		// If we get any error, the connection very probably died. Even if that's not the case,
		// we should stop the client and give the reason for the shutdown
		if err != nil {
			go func() { _ = s.close(fmt.Errorf("read loop died: %w", util.ParseGrpcError(err))) }()
			return
		}

		// Try to send the response back to the requester (if any)
		if s.handleResponse(msg) {
			continue
		}

		// Otherwise, deliver the message to the message channel
		select {
		case s.messages <- msg:
		case <-s.closed:
			return
		}

	}

}

// handleResponse handles a response message and returns true if it was handled.
func (s *Session) handleResponse(msg *proto.ShellServerMessage) bool {

	// Check if the message is a valid response
	if msg.Id == "" {
		return false
	}

	// Lock the pending messages map
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to deliver the message to the pending channel
	if s.pending != nil {
		if ch, ok := s.pending[msg.Id]; ok {
			delete(s.pending, msg.Id)
			ch <- msg
			return true
		}
	}

	// If we reach here, the caller gave up waiting for a response (context canceled after the message was sent)
	// Return true so readLoop doesn't try to deliver it to the messages channel.
	return true

}

// removePending removes a pending message from the list with thread safety.
func (s *Session) removePending(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
}

// registerPending adds a pending message with thread safety.
func (s *Session) registerPending(id string) chan *proto.ShellServerMessage {
	respCh := make(chan *proto.ShellServerMessage, 1)
	s.mu.Lock()
	s.pending[id] = respCh
	s.mu.Unlock()
	return respCh
}
