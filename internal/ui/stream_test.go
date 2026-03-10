package ui

import "testing"

func TestThinkFilter_BasicBlock(t *testing.T) {
	f := &thinkFilter{}
	got := f.Filter("<think>reasoning here</think>Hello world")
	got += f.Flush()
	if got != "Hello world" {
		t.Errorf("got %q, want %q", got, "Hello world")
	}
}

func TestThinkFilter_StreamingChunks(t *testing.T) {
	f := &thinkFilter{}
	var out string
	out += f.Filter("<thi")
	out += f.Filter("nk>internal ")
	out += f.Filter("reasoning</th")
	out += f.Filter("ink>visible content")
	out += f.Flush()
	if out != "visible content" {
		t.Errorf("got %q, want %q", out, "visible content")
	}
}

func TestThinkFilter_NoThinkTags(t *testing.T) {
	f := &thinkFilter{}
	got := f.Filter("plain text without think tags")
	got += f.Flush()
	if got != "plain text without think tags" {
		t.Errorf("got %q, want %q", got, "plain text without think tags")
	}
}

func TestThinkFilter_MultipleBlocks(t *testing.T) {
	f := &thinkFilter{}
	got := f.Filter("<think>first</think>A<think>second</think>B")
	got += f.Flush()
	if got != "AB" {
		t.Errorf("got %q, want %q", got, "AB")
	}
}

func TestThinkFilter_OnlyThink(t *testing.T) {
	f := &thinkFilter{}
	got := f.Filter("<think>only reasoning here</think>")
	got += f.Flush()
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}
