package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	pb "github.com/bluefunda/cai-cli/api/proto/bff"
	"google.golang.org/grpc"
)

// thinkFilter strips <think>...</think> blocks from streamed content.
// It handles tags that span multiple chunks.
type thinkFilter struct {
	inside bool   // true when inside a <think> block
	buf    string // partial tag buffer
}

func (f *thinkFilter) Filter(chunk string) string {
	f.buf += chunk
	var out strings.Builder

	for len(f.buf) > 0 {
		if f.inside {
			idx := strings.Index(f.buf, "</think>")
			if idx >= 0 {
				f.buf = f.buf[idx+len("</think>"):]
				f.inside = false
				continue
			}
			// Could be a partial </think> at the end — keep the tail.
			if partialLen := partialSuffix(f.buf, "</think>"); partialLen > 0 {
				f.buf = f.buf[len(f.buf)-partialLen:]
			} else {
				f.buf = ""
			}
			return out.String()
		}

		idx := strings.Index(f.buf, "<think>")
		if idx >= 0 {
			out.WriteString(f.buf[:idx])
			f.buf = f.buf[idx+len("<think>"):]
			f.inside = true
			continue
		}
		// Check for a partial <think> at the end of the buffer.
		if partialLen := partialSuffix(f.buf, "<think>"); partialLen > 0 {
			out.WriteString(f.buf[:len(f.buf)-partialLen])
			f.buf = f.buf[len(f.buf)-partialLen:]
			return out.String()
		}
		out.WriteString(f.buf)
		f.buf = ""
	}

	return out.String()
}

// Flush returns any remaining buffered content (partial tag that never completed).
func (f *thinkFilter) Flush() string {
	s := f.buf
	f.buf = ""
	if f.inside {
		return ""
	}
	return s
}

// partialSuffix returns the length of the longest suffix of s that is a prefix of tag.
func partialSuffix(s, tag string) int {
	maxLen := len(tag) - 1
	if maxLen > len(s) {
		maxLen = len(s)
	}
	for n := maxLen; n > 0; n-- {
		if strings.HasSuffix(s, tag[:n]) {
			return n
		}
	}
	return 0
}

// RenderGRPCStream reads ChatEvent messages from a gRPC server stream,
// prints content chunks to stdout, and calls cancelFn on Ctrl+C.
func RenderGRPCStream(stream grpc.ServerStreamingClient[pb.ChatEvent], cancelFn context.CancelFunc) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	doneCh := make(chan error, 1)

	go func() {
		doneCh <- renderGRPCLoop(stream)
	}()

	select {
	case err := <-doneCh:
		return err
	case <-sigCh:
		fmt.Println()
		cancelFn()
		return nil
	}
}

func renderGRPCLoop(stream grpc.ServerStreamingClient[pb.ChatEvent]) error {
	tf := &thinkFilter{}
	for {
		ev, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Print(tf.Flush())
				fmt.Println()
				return nil
			}
			return fmt.Errorf("stream recv: %w", err)
		}

		switch ev.GetType() {
		case "content", "stream_chunk":
			fmt.Print(tf.Filter(ev.GetContent()))
		case "error", "stream_error":
			Error(ev.GetError())
		case "done", "stream_end":
			fmt.Print(tf.Flush())
			fmt.Println()
			return nil
		case "tool_call":
			Info(fmt.Sprintf("Tool call: %s", ev.GetData()))
		case "tool_result", "stream_start", "stream_heartbeat":
			// No display needed.
		default:
			// Unknown event type; ignore.
		}
	}
}
