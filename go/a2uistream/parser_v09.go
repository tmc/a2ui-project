package a2uistream

import (
	"encoding/json"
	"io"

	"github.com/a2ui-project/a2ui/go/a2ui/v09"
)

// ResponsePartV09 is like [ResponsePart] for A2UI v0.9.
// Messages holds the v0.9 and v0.9.x messages in Payload, decoded.
type ResponsePartV09 struct {
	Text     string              // conversational text
	Messages []v09.ServerMessage // A2UI v0.9 and v0.9.x messages (nil if none)
	Payload  []map[string]any    // A2UI messages of any version (nil if text-only)
}

// ParserV09 is like [Parser] but returns [ResponsePartV09] values.
type ParserV09 struct {
	p Parser
}

// NewParserV09 creates a new streaming parser for A2UI v0.9 messages.
func NewParserV09() *ParserV09 {
	return &ParserV09{}
}

// ProcessChunk feeds a chunk of text and returns any complete parts found.
func (p *ParserV09) ProcessChunk(chunk string) ([]ResponsePartV09, error) {
	parts, err := p.p.ProcessChunk(chunk)
	return partsV09(parts), err
}

// Flush returns any remaining buffered content as a text part.
func (p *ParserV09) Flush() ([]ResponsePartV09, error) {
	parts, err := p.p.Flush()
	return partsV09(parts), err
}

// Reset clears all parser state for reuse.
func (p *ParserV09) Reset() {
	p.p.Reset()
}

// ReaderV09 is like [Reader] but returns [ResponsePartV09] values.
type ReaderV09 struct {
	r *Reader
}

// NewReaderV09 creates a v0.9 parser that reads from r.
func NewReaderV09(r io.Reader) *ReaderV09 {
	return &ReaderV09{r: NewReader(r)}
}

// Next returns the next parsed response part.
func (r *ReaderV09) Next() (ResponsePartV09, error) {
	part, err := r.r.Next()
	if err != nil {
		return ResponsePartV09{}, err
	}
	return partV09(part), nil
}

func partsV09(parts []ResponsePart) []ResponsePartV09 {
	if parts == nil {
		return nil
	}
	out := make([]ResponsePartV09, len(parts))
	for i, part := range parts {
		out[i] = partV09(part)
	}
	return out
}

// partV09 decodes the v0.9.x messages in part.Payload.
// As with 1.x messages in [ResponsePart], a payload that does not decode
// is kept in Payload but left out of Messages.
func partV09(part ResponsePart) ResponsePartV09 {
	out := ResponsePartV09{Text: part.Text, Payload: part.Payload}
	for _, payload := range part.Payload {
		if !isV09(payloadVersion(payload)) {
			continue
		}
		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		var msg v09.ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		out.Messages = append(out.Messages, msg)
	}
	return out
}
