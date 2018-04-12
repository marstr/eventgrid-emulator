package model

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/eventgrid"
	"github.com/Azure/go-autorest/autorest"
)

var callbackRetryDuration = 24 * time.Hour

func CallbackRetryDuration() time.Duration {
	return callbackRetryDuration
}

func SetCallbackRetryDuration(val time.Duration) {
	callbackRetryDuration = val
}

// ProcessEvent takes an event, and sends a message to each of the callbacks
// registered that matches the event.
func ProcessEvent(ctx context.Context, event eventgrid.Event) error {
	return ProcessEventList(ctx, event, ListFilteredSubscribers(event))
}

// ProcessEventList iterates over the list of callbacks and fires a Request with each
func ProcessEventList(ctx context.Context, event eventgrid.Event, callbacks []string) (err error) {
	var body []byte
	body, err = json.Marshal(event)
	if err != nil {
		return
	}

	client := &autorest.Client{}
	client.AddToUserAgent("eventgrid-emulator")
	client.AddToUserAgent(runtime.Version())

	sender := autorest.DecorateSender(client, autorest.DoRetryForDuration(CallbackRetryDuration(), time.Second))

	var bodyReader io.ReadSeeker
	bodyReader = bytes.NewReader(body)

	for _, callback := range callbacks {
		if _, ok := <-ctx.Done(); !ok {
			return ctx.Err()
		}

		var req *http.Request
		req, err = http.NewRequest(http.MethodPost, callback, bodyReader)
		if err != nil {
			return
		}
		req = req.WithContext(ctx)

		_, err = sender.Do(req)
		if err != nil {
			return
		}

		bodyReader.Seek(0, io.SeekStart)
	}

	return
}
