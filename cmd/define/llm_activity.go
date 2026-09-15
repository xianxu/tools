package main

import (
	"context"
	"io"

	"github.com/xianxu/tools/internal/llm"
)

// activityClient owns only foreground waiting. clientModelSelection unwraps it
// when a shared caller needs the underlying transport's provenance.
type activityClient struct {
	client llm.Client
	host   activityHost
}

func clientModelSelection(client llm.Client) llm.ModelSelection {
	for {
		activity, ok := client.(*activityClient)
		if !ok {
			return llm.SelectionOf(client)
		}
		client = activity.client
	}
}

// foregroundClient chooses the display before answer writers wrap stdout.
// Background work receives the original client and owns its own display policy.
func foregroundClient(client llm.Client, out io.Writer, opt options) llm.Client {
	if !activityEnabled(opt) {
		return client
	}
	var host activityHost = &plainActivityHost{out: out}
	if screen, ok := out.(*liveScreen); ok {
		host = &screenActivityHost{screen: screen}
	}
	return &activityClient{client: client, host: host}
}

func (c *activityClient) Complete(ctx context.Context, r llm.Request) (llm.Response, error) {
	stop := startActivity(ctx, c.host)
	defer stop()
	return c.client.Complete(ctx, r)
}

func (c *activityClient) Stream(ctx context.Context, r llm.Request, onDelta func(string)) (llm.Response, error) {
	stop := startActivity(ctx, c.host)
	defer stop()
	return c.client.Stream(ctx, r, func(s string) {
		if s != "" {
			stop()
		}
		if onDelta != nil {
			onDelta(s)
		}
	})
}
