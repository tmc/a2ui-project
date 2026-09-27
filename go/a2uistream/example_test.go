package a2uistream_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/a2ui-project/a2ui/go/a2uistream"
)

func ExampleReader_Next() {
	input := `Before <a2ui-json>{"version":"v1.0","deleteSurface":{"surfaceId":"old"}}</a2ui-json> after`
	r := a2uistream.NewReader(strings.NewReader(input))
	for {
		part, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if part.Text != "" {
			fmt.Println(strings.TrimSpace(part.Text))
		}
		for _, msg := range part.Messages {
			fmt.Println(msg.DeleteSurface.SurfaceID)
		}
	}
	// Output:
	// Before
	// old
	// after
}

func ExampleReader_Next_payload() {
	input := `Before <a2ui-json>{"version":"v1.0","callRendererFunction":{"functionCallId":"call-1","callFunction":{"call":"lookup","catalogId":"https://example.com/catalog.json"}}}</a2ui-json>`
	r := a2uistream.NewReader(strings.NewReader(input))
	for {
		part, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		for _, payload := range part.Payload {
			fmt.Println(payload["version"])
			call := payload["callRendererFunction"].(map[string]any)
			fmt.Println(call["functionCallId"])
		}
	}
	// Output:
	// v1.0
	// call-1
}

func ExampleParseAndValidate() {
	_, err := a2uistream.ParseAndValidate(`<a2ui-json>{"version":"v0.9","deleteSurface":{"surfaceId":"old"}}</a2ui-json>`, nil)
	fmt.Println(err)

	parts, err := a2uistream.ParseAndValidate(`<a2ui-json>{"version":"v1.0","deleteSurface":{"surfaceId":"old"}}</a2ui-json>`, nil)
	fmt.Println(parts[0].Messages[0].DeleteSurface.SurfaceID, err)
	// Output:
	// a2uistream: message version "v0.9" is not 1.x
	// old <nil>
}

func ExampleFixPayload() {
	payload, err := a2uistream.FixPayload(`{"type": “Text”, "text": "Hello",}`)
	if err != nil {
		panic(err)
	}
	fmt.Println(payload[0]["type"])
	fmt.Println(payload[0]["text"])
	// Output:
	// Text
	// Hello
}

func ExampleParseResponse() {
	parts, err := a2uistream.ParseResponse(`Intro
<a2ui-json>[{"id":"card"}]</a2ui-json>
Done`)
	if err != nil {
		panic(err)
	}
	fmt.Println(parts[0].Text)
	fmt.Println(parts[0].Payload[0]["id"])
	fmt.Println(parts[1].Text)
	// Output:
	// Intro
	// card
	// Done
}
