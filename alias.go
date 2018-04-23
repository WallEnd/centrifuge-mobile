package centrifuge

import "github.com/centrifugal/centrifuge-mobile/proto"

// Error represents client reply error.
type Error = proto.Error

// Raw represents raw bytes.
type Raw = proto.Raw

// Pub allows to deliver custom payload to all channel subscribers.
type Pub = proto.Pub

// Join sent to channel after someone subscribed.
type Join = proto.Join

// Leave sent to channel after someone unsubscribed.
type Leave = proto.Leave

// Unsub sent to channel when server wants to unsubscribe client.
type Unsub = proto.Unsub

// ClientInfo is short information about client connection.
type ClientInfo = proto.ClientInfo

// Encoding represents client connection transport encoding format.
type Encoding = proto.Encoding
