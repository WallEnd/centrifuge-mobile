package centrifuge

import (
	"sync"
	"time"

	"github.com/centrifugal/centrifuge-mobile/internal/proto"
)

// SubscribeSuccessEvent is a subscribe success event context passed to event callback.
type SubscribeSuccessEvent struct {
	Resubscribed bool
	Recovered    bool
}

// SubscribeErrorEvent is a subscribe error event context passed to event callback.
type SubscribeErrorEvent struct {
	Error string
}

// UnsubscribeEvent is a context passed to unsubscribe event callback.
type UnsubscribeEvent struct{}

// LeaveEvent ...
type LeaveEvent struct {
	ClientInfo
}

// JoinEvent ...
type JoinEvent struct {
	ClientInfo
}

// PublicationEvent ...
type PublicationEvent struct {
	Publication
}

// PublicationHandler is a function to handle messages published in channels.
type PublicationHandler interface {
	OnPublication(*Sub, PublicationEvent)
}

// JoinHandler is a function to handle join messages.
type JoinHandler interface {
	OnJoin(*Sub, JoinEvent)
}

// LeaveHandler is a function to handle leave messages.
type LeaveHandler interface {
	OnLeave(*Sub, LeaveEvent)
}

// UnsubscribeHandler is a function to handle unsubscribe event.
type UnsubscribeHandler interface {
	OnUnsubscribe(*Sub, UnsubscribeEvent)
}

// SubscribeSuccessHandler is a function to handle subscribe success event.
type SubscribeSuccessHandler interface {
	OnSubscribeSuccess(*Sub, SubscribeSuccessEvent)
}

// SubscribeErrorHandler is a function to handle subscribe error event.
type SubscribeErrorHandler interface {
	OnSubscribeError(*Sub, SubscribeErrorEvent)
}

// SubEventHandler contains callback functions that will be called when
// corresponding event happens with subscription to channel.
type SubEventHandler struct {
	onPublication      PublicationHandler
	onJoin             JoinHandler
	onLeave            LeaveHandler
	onUnsubscribe      UnsubscribeHandler
	onSubscribeSuccess SubscribeSuccessHandler
	onSubscribeError   SubscribeErrorHandler
}

// NewSubEventHandler initializes new SubEventHandler.
func NewSubEventHandler() *SubEventHandler {
	return &SubEventHandler{}
}

// OnPublication allows to set PublicationHandler to SubEventHandler.
func (h *SubEventHandler) OnPublication(handler PublicationHandler) {
	h.onPublication = handler
}

// OnJoin allows to set JoinHandler to SubEventHandler.
func (h *SubEventHandler) OnJoin(handler JoinHandler) {
	h.onJoin = handler
}

// OnLeave allows to set LeaveHandler to SubEventHandler.
func (h *SubEventHandler) OnLeave(handler LeaveHandler) {
	h.onLeave = handler
}

// OnUnsubscribe allows to set UnsubscribeHandler to SubEventHandler.
func (h *SubEventHandler) OnUnsubscribe(handler UnsubscribeHandler) {
	h.onUnsubscribe = handler
}

// OnSubscribeSuccess allows to set SubscribeSuccessHandler to SubEventHandler.
func (h *SubEventHandler) OnSubscribeSuccess(handler SubscribeSuccessHandler) {
	h.onSubscribeSuccess = handler
}

// OnSubscribeError allows to set SubscribeErrorHandler to SubEventHandler.
func (h *SubEventHandler) OnSubscribeError(handler SubscribeErrorHandler) {
	h.onSubscribeError = handler
}

// Describe different states of Sub.
const (
	NEW = iota
	SUBSCRIBING
	SUBSCRIBED
	SUBERROR
	UNSUBSCRIBED
)

// Sub describes client subscription to channel.
type Sub struct {
	mu              sync.Mutex
	channel         string
	centrifuge      *Client
	status          int
	events          *SubEventHandler
	lastMessageID   *string
	lastMessageMu   sync.RWMutex
	resubscribed    bool
	recovered       bool
	err             error
	needResubscribe bool
	subFutures      []chan error
}

func (c *Client) newSub(channel string, events *SubEventHandler) *Sub {
	s := &Sub{
		centrifuge:      c,
		channel:         channel,
		events:          events,
		subFutures:      make([]chan error, 0),
		needResubscribe: true,
	}
	return s
}

// Channel returns subscription channel.
func (s *Sub) Channel() string {
	return s.channel
}

func (s *Sub) newSubFuture() chan error {
	fut := make(chan error, 1)
	s.mu.Lock()
	if s.status == SUBSCRIBED {
		fut <- nil
	} else if s.status == SUBERROR {
		fut <- s.err
	} else {
		s.subFutures = append(s.subFutures, fut)
	}
	s.mu.Unlock()
	return fut
}

// Sub.mu lock must be held outside.
func (s *Sub) resolveSubFutures(err error) {
	for _, ch := range s.subFutures {
		select {
		case ch <- err:
		default:
		}
	}
	s.subFutures = nil
}

func (s *Sub) removeSubFuture(subFuture chan error) {
	s.mu.Lock()
	for i, v := range s.subFutures {
		if v == subFuture {
			s.subFutures = append(s.subFutures[:i], s.subFutures[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
}

// Publish allows to publish data to channel.
func (s *Sub) Publish(data []byte) error {
	subFuture := s.newSubFuture()
	select {
	case err := <-subFuture:
		if err != nil {
			return err
		}
		return s.centrifuge.publish(s.channel, data)
	case <-time.After(time.Duration(s.centrifuge.config.TimeoutMilliseconds) * time.Millisecond):
		s.removeSubFuture(subFuture)
		return ErrTimeout
	}
}

func (s *Sub) history() ([]Publication, error) {
	subFuture := s.newSubFuture()
	select {
	case err := <-subFuture:
		if err != nil {
			return nil, err
		}
		return s.centrifuge.history(s.channel)
	case <-time.After(time.Duration(s.centrifuge.config.TimeoutMilliseconds) * time.Millisecond):
		s.removeSubFuture(subFuture)
		return nil, ErrTimeout
	}
}

func (s *Sub) presence() (map[string]proto.ClientInfo, error) {
	subFuture := s.newSubFuture()
	select {
	case err := <-subFuture:
		if err != nil {
			return nil, err
		}
		return s.centrifuge.presence(s.channel)
	case <-time.After(time.Duration(s.centrifuge.config.TimeoutMilliseconds) * time.Millisecond):
		s.removeSubFuture(subFuture)
		return nil, ErrTimeout
	}
}

// Unsubscribe allows to unsubscribe from channel.
func (s *Sub) Unsubscribe() error {
	s.centrifuge.unsubscribe(s.channel)
	s.triggerOnUnsubscribe(false)
	return nil
}

// Subscribe allows to subscribe again after unsubscribing.
func (s *Sub) Subscribe() error {
	s.mu.Lock()
	s.needResubscribe = true
	s.mu.Unlock()
	return s.resubscribe()
}

func (s *Sub) triggerOnUnsubscribe(needResubscribe bool) {
	s.mu.Lock()
	if s.status != SUBSCRIBED {
		s.mu.Unlock()
		return
	}
	s.needResubscribe = needResubscribe
	s.status = UNSUBSCRIBED
	s.mu.Unlock()
	if s.events != nil && s.events.onUnsubscribe != nil {
		handler := s.events.onUnsubscribe
		handler.OnUnsubscribe(s, UnsubscribeEvent{})
	}
}

func (s *Sub) subscribeSuccess(recovered bool) {
	s.mu.Lock()
	if s.status == SUBSCRIBED {
		s.mu.Unlock()
		return
	}
	s.status = SUBSCRIBED
	resubscribed := s.resubscribed
	s.resolveSubFutures(nil)
	s.mu.Unlock()
	if s.events != nil && s.events.onSubscribeSuccess != nil {
		handler := s.events.onSubscribeSuccess
		handler.OnSubscribeSuccess(s, SubscribeSuccessEvent{Resubscribed: resubscribed, Recovered: recovered})
	}
	s.mu.Lock()
	s.resubscribed = true
	s.mu.Unlock()
}

func (s *Sub) subscribeError(err error) {
	s.mu.Lock()
	if s.status == SUBERROR {
		s.mu.Unlock()
		return
	}
	s.err = err
	s.status = SUBERROR
	s.resolveSubFutures(err)
	s.mu.Unlock()
	if s.events != nil && s.events.onSubscribeError != nil {
		handler := s.events.onSubscribeError
		handler.OnSubscribeError(s, SubscribeErrorEvent{Error: err.Error()})
	}
}

func (s *Sub) handlePub(pub Publication) {
	var handler PublicationHandler
	if s.events != nil && s.events.onPublication != nil {
		handler = s.events.onPublication
	}
	mid := pub.UID
	s.lastMessageMu.Lock()
	s.lastMessageID = &mid
	s.lastMessageMu.Unlock()
	if handler != nil {
		handler.OnPublication(s, PublicationEvent{Publication: pub})
	}
}

func (s *Sub) handleJoin(info proto.ClientInfo) {
	var handler JoinHandler
	if s.events != nil && s.events.onJoin != nil {
		handler = s.events.onJoin
	}
	if handler != nil {
		handler.OnJoin(s, JoinEvent{ClientInfo: info})
	}
}

func (s *Sub) handleLeave(info proto.ClientInfo) {
	var handler LeaveHandler
	if s.events != nil && s.events.onLeave != nil {
		handler = s.events.onLeave
	}
	if handler != nil {
		handler.OnLeave(s, LeaveEvent{ClientInfo: info})
	}
}

func (s *Sub) handleUnsub(m proto.Unsub) {
	s.Unsubscribe()
}

func (s *Sub) resubscribe() error {
	s.mu.Lock()
	if s.status == SUBSCRIBED || s.status == SUBSCRIBING {
		s.mu.Unlock()
		return nil
	}
	needResubscribe := s.needResubscribe
	s.mu.Unlock()
	if !needResubscribe {
		return nil
	}

	s.centrifuge.mutex.Lock()
	if s.centrifuge.status != CONNECTED {
		s.centrifuge.mutex.Unlock()
		return nil
	}
	s.centrifuge.mutex.Unlock()

	s.mu.Lock()
	s.status = SUBSCRIBING
	s.mu.Unlock()

	privateSign, err := s.centrifuge.privateSign(s.channel)
	if err != nil {
		return err
	}

	var msgID *string
	s.lastMessageMu.Lock()
	if s.lastMessageID != nil {
		msg := *s.lastMessageID
		msgID = &msg
	}
	s.lastMessageMu.Unlock()
	res, err := s.centrifuge.sendSubscribe(s.channel, msgID, privateSign)
	if err != nil {
		if err == ErrTimeout {
			return err
		}
		s.subscribeError(err)
		return nil
	}

	if len(res.Publications) > 0 {
		for i := len(res.Publications) - 1; i >= 0; i-- {
			s.handlePub(*res.Publications[i])
		}
	} else {
		lastID := string(res.Last)
		s.lastMessageMu.Lock()
		s.lastMessageID = &lastID
		s.lastMessageMu.Unlock()
	}

	s.subscribeSuccess(res.Recovered)
	return nil
}
