package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	pb "github.com/bluefunda/cai-cli/api/proto/bff"
	"google.golang.org/grpc"
)

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
	for {
		ev, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return fmt.Errorf("stream recv: %w", err)
		}

		switch ev.GetType() {
		case "content", "stream_chunk":
			fmt.Print(ev.GetContent())
		case "error", "stream_error":
			Error(ev.GetError())
		case "done", "stream_end":
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
