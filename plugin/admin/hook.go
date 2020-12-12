package admin

import (
	"context"

	"github.com/DrmagicE/gmqtt"
	"github.com/DrmagicE/gmqtt/server"
)

func (a *Admin) OnSessionCreatedWrapper(pre server.OnSessionCreated) server.OnSessionCreated {
	return func(ctx context.Context, client server.Client) {
		pre(ctx, client)
		a.store.addClient(client)
	}
}

func (a *Admin) OnSessionResumeWrapper(pre server.OnSessionResumed) server.OnSessionResumed {
	return func(ctx context.Context, client server.Client) {
		pre(ctx, client)
		a.store.addClient(client)
	}
}
func (a *Admin) OnSessionTerminatedWrapper(pre server.OnSessionTerminated) server.OnSessionTerminated {
	return func(ctx context.Context, clientID string, reason server.SessionTerminatedReason) {
		pre(ctx, clientID, reason)
		a.store.removeClient(clientID)
	}
}

func (a *Admin) OnCloseWrapper(pre server.OnClose) server.OnClose {
	return func(ctx context.Context, client server.Client, err error) {
		pre(ctx, client, err)
		a.store.setClientDisconnected(client.ClientOptions().ClientID)
	}
}

func (a *Admin) OnSubscribedWrapper(pre server.OnSubscribed) server.OnSubscribed {
	return func(ctx context.Context, client server.Client, subscription *gmqtt.Subscription) {
		pre(ctx, client, subscription)
		a.store.addSubscription(client.ClientOptions().ClientID, subscription)
	}
}

func (a *Admin) OnUnsubscribedWrapper(pre server.OnUnsubscribed) server.OnUnsubscribed {
	return func(ctx context.Context, client server.Client, topicName string) {
		pre(ctx, client, topicName)
		a.store.removeSubscription(client.ClientOptions().ClientID, topicName)
	}
}