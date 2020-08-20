package tss

import "github.com/keep-network/keep-core/pkg/net"

// TSSProtocolMessage is a network message used to transport messages generated in
// TSS protocol execution. It is a wrapper over a message generated by underlying
// implementation of the protocol.
type TSSProtocolMessage struct {
	SenderID    MemberID
	Payload     []byte
	IsBroadcast bool
	SessionID   string
}

// Type returns a string type of the `TSSMessage` so that it conforms to
// `net.Message` interface.
func (m *TSSProtocolMessage) Type() string {
	return "ecdsa/tss_message"
}

// ReadyMessage is a network message used to notify peer members about readiness
// to start protocol execution.
type ReadyMessage struct {
	SenderID MemberID
}

// Type returns a string type of the `ReadyMessage`.
func (m *ReadyMessage) Type() string {
	return "ecdsa/ready_message"
}

// AnnounceMessage is a network message used to announce peer's presence.
type AnnounceMessage struct {
	SenderID MemberID
}

// Type returns a string type of the `AnnounceMessage`.
func (m *AnnounceMessage) Type() string {
	return "ecdsa/announce_message"
}

func RegisterUnmarshalers(broadcastChannel net.BroadcastChannel) {
	err := broadcastChannel.RegisterUnmarshaler(func() net.TaggedUnmarshaler {
		return &AnnounceMessage{}
	})
	if err != nil {
		logger.Errorf(
			"could not register AnnounceMessage unmarshaller: [%v]",
			err,
		)
	}

	err = broadcastChannel.RegisterUnmarshaler(func() net.TaggedUnmarshaler {
		return &ReadyMessage{}
	})
	if err != nil {
		logger.Errorf(
			"could not register ReadyMessage unmarshaller: [%v]",
			err,
		)
	}

	err = broadcastChannel.RegisterUnmarshaler(func() net.TaggedUnmarshaler {
		return &TSSProtocolMessage{}
	})
	if err != nil {
		logger.Errorf(
			"could not register TSSProtocolMessage unmarshaller: [%v]",
			err,
		)
	}
}
